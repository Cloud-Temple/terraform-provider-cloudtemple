package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Anti-orphan recovery for the asynchronous VMware virtual machine create paths (#527).
//
// Every create mode (clone, content library, marketplace, from scratch) answers
// 202 + an activity id: FROM THAT INSTANT A VM MAY EXIST PLATFORM-SIDE. The wait
// that follows can then fail for reasons that say nothing about whether the VM
// was created — an activity that freezes and never completes, a transient
// network blip while polling, a cancelled apply.
//
// Returning that failure with an empty resource id makes SDKv2 record NOTHING:
// ResourceData.State() returns nil on an empty id, so Terraform stores no
// object at all. The VM is then ORPHANED — `terraform destroy` cannot reach it,
// `terraform apply` cannot converge it, and the only way out is a manual
// deletion in the console.
//
// The platform publishes the created VM in the activity's concernedItems as
// soon as it is materialised, WHILE THE ACTIVITY IS STILL RUNNING. Measured
// against the API on a marketplace VMware deployment: the `virtual_machine`
// concerned item appears about 8 seconds after the POST, roughly a minute
// before `state.completed.result` carries that same id.
//
// That signal is not merely the most convenient one, it is the ONLY SOUND one.
// It is scoped to THIS activity, whose id came from the Location header of
// THIS resource's own POST, so concurrent creates cannot contaminate each
// other. Every alternative is a GLOBAL lookup, and no global lookup can be
// made safe, because nothing on a virtual machine records which activity
// created it:
//
//   - matching by name adopts a homonym the provider never created;
//   - a differential (census the name before the POST, adopt what is new
//     afterwards) fails under concurrency, which is exactly when this bug
//     shows up. Five instances created in parallel under one name: instance A
//     censuses zero, instance B's machine materialises, A fails and finds
//     exactly ONE machine that is new since its census — B's. A adopts it,
//     taints it, and the next apply destroys a machine B also owns. An
//     "exactly one candidate" rule does not help: the race produces exactly
//     one candidate, just not the right one.
//
// So when the activity names no virtual machine, this code adopts NOTHING and
// says what to audit. That is a deliberate refusal, not a missing feature.
//
// Adopting an id is a STATE-CRITICAL decision: the adopted resource is
// persisted as TAINTED, so the next apply DESTROYS whatever it points at. The
// evidence required is therefore strict, and every ambiguous case fails closed
// with a diagnostic that tells the operator exactly what to audit.
//
// Why TAINTED and not "recorded as healthy": a create that returns an error
// leaves SDKv2 exactly two outcomes — no id (the object is orphaned) or an id
// (the object is tainted). There is no third option, and the provider cannot
// know at failure time whether an object whose activity never completed will
// end up complete. Tainted is the safe default AND the reversible one: when the
// operator verifies the virtual machine is actually complete, `terraform
// untaint` keeps it and a re-apply converges the configuration instead of
// recreating it. The diagnostics say so, because "the deployment merely stalled
// and the machine is fine" is a plausible reading of the failure the operator
// must be able to act on.

// vmReadFunc abstracts the per-id virtual machine read so the recovery is unit
// tested without HTTP calls. It follows the client contract: (nil, nil) is a
// definitive 404, (nil, err) is an inconclusive read.
type vmReadFunc func(ctx context.Context, id string) (*client.VirtualMachine, error)

// vmCandidateFromActivity returns the id of the virtual machine this activity
// created, and ONLY when the evidence is unambiguous: exactly one concerned
// item of type "virtual_machine" once the ids known to designate something else
// are excluded.
//
// The exclusion set carries the ids the caller already knows are NOT the new
// VM — on the clone path, the source virtual machine. Whether the platform
// lists the clone source as a concerned item is not something the provider can
// rely on, so the exclusion makes the answer correct either way: if the source
// is listed it is removed, and if it is not, removing it changes nothing.
//
// Zero candidates (the VM was not materialised, or the activity only ever
// referenced other objects) and several candidates (an activity shape this code
// cannot interpret) BOTH return "". Adopting under ambiguity would risk
// tainting — and therefore destroying — a virtual machine this resource never
// created, which is strictly worse than the orphan it would avoid.
func vmCandidateFromActivity(activity *client.Activity, exclude ...string) string {
	if activity == nil {
		return ""
	}

	excluded := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		if id != "" {
			excluded[id] = struct{}{}
		}
	}

	var candidate string
	for _, item := range activity.ConcernedItems {
		if item.Type != "virtual_machine" || item.ID == "" {
			continue
		}
		if _, skip := excluded[item.ID]; skip {
			continue
		}
		if candidate != "" && candidate != item.ID {
			// Several distinct virtual machines: no unambiguous winner.
			return ""
		}
		candidate = item.ID
	}
	return candidate
}

// recoverVMCreateFailure is the failure path shared by every asynchronous
// VMware virtual machine create. It turns a bare error into a decision about
// what the Terraform state must hold, and returns the diagnostic to surface.
//
// action is the human-readable operation that failed ("failed to deploy
// marketplace item"), name the virtual machine name that was requested, and
// cause the underlying error. exclude carries ids that are known not to be the
// created VM (the clone source).
//
// The four outcomes, in order of decreasing evidence:
//
//  1. An id is already set — the activity completed far enough for
//     state.result, or the caller resolved it. The resource is tracked; only
//     report the failure.
//  2. Exactly one virtual machine concerned item, confirmed by a read-back to
//     exist, to NOT be a template, and to carry the requested name: adopt it,
//     and record machine_manager_id so a later refresh can resolve the resource
//     (computeVirtualMachineRead fails closed without it). A template or a name
//     mismatch means the activity designated some other object — adopting it
//     would taint, and on the next apply destroy, something this resource never
//     created — so both fail closed.
//  3. Exactly one concerned item whose read-back is INCONCLUSIVE (transport or
//     server error): adopt it anyway. The reported failure mode is precisely a
//     virtual machine that outlives the apply, and refusing to track it on a
//     read that proves nothing is the orphan this code exists to prevent.
//  4. No candidate, or a read-back that DEFINITIVELY proves the VM is gone
//     (404): record nothing, and say what to audit. Adopting an id that no
//     longer resolves would be worse than recording nothing: the read fails
//     closed and never auto-removes, so every later refresh would error until
//     the operator ran `terraform state rm`.
//
// vmAbsenceAttempts is the number of ADDITIONAL read-backs used before concluding
// that a candidate virtual machine really does not exist, and vmAbsenceBackoff the
// pause between them.
//
// A single 404 is not proof of absence on this platform. The virtual machine item is
// published on the activity as soon as the object is materialised — roughly eight
// seconds after the POST, measured — while `GET /compute/v1/vcenters/virtual_machines/{id}`
// has documented eventual-consistency windows right after a write (#415). Read-back
// and indexing can therefore cross: the object exists and the id still answers 404.
//
// Concluding absence from that single answer is the expensive mistake. The two
// outcomes are not symmetric:
//   - wrongly concluding absence records nothing, and the operator is sent away while
//     a billable virtual machine materialises unattended — the #527 damage itself;
//   - wrongly concluding presence adopts an id, which the case-4 reasoning explains is
//     also bad (the read fails closed and never auto-removes).
//
// So neither answer is safe on ONE read, and the cheapest way out is simply to look
// again. Two extra reads cost a few seconds on a path that has already failed, and
// they are by id — no listing, no wide call.
const vmAbsenceAttempts = 2

// vmAbsenceBackoff is a var, not a const, solely so tests can drive the absence path
// without sleeping. Production never reassigns it.
var vmAbsenceBackoff = 3 * time.Second

// confirmVMAbsence re-reads a candidate that answered "not found" once.
//
// It returns as soon as anything contradicts the absence: a virtual machine (it
// exists after all — adopt it) or an error (inconclusive — the caller adopts rather
// than orphan). Only a run of definitive not-founds returns (nil, nil), which is the
// answer the caller is allowed to treat as proof.
func confirmVMAbsence(ctx context.Context, read vmReadFunc, candidate string) (*client.VirtualMachine, error) {
	for attempt := 0; attempt < vmAbsenceAttempts; attempt++ {
		select {
		case <-ctx.Done():
			// A cancelled context proves nothing about the platform. Report it as an
			// error so the caller takes the INCONCLUSIVE branch and adopts, rather
			// than orphaning on a read that never happened.
			return nil, ctx.Err()
		case <-time.After(vmAbsenceBackoff):
		}
		vm, err := read(ctx, candidate)
		if err != nil || vm != nil {
			return vm, err
		}
	}
	return nil, nil
}

func recoverVMCreateFailure(
	ctx context.Context,
	d *schema.ResourceData,
	read vmReadFunc,
	activity *client.Activity,
	activityID string,
	name string,
	action string,
	cause error,
	exclude ...string,
) diag.Diagnostics {
	if d.Id() != "" {
		return diag.Errorf(
			"%s: %s. The virtual machine %s IS recorded in the Terraform state (tainted): run `terraform destroy` to remove it, or `terraform apply` to replace it — no manual cleanup is needed. If you verify the virtual machine is actually complete, `terraform untaint` it and re-apply to finish the configuration instead of recreating it.",
			action, cause, d.Id(),
		)
	}

	candidate := vmCandidateFromActivity(activity, exclude...)
	if candidate == "" {
		return diag.Errorf(
			"%s: %s. Activity %s did not designate a single virtual machine, so NOTHING was recorded in the Terraform state. If a virtual machine named %q was created it is ORPHANED — audit it and either import it (`terraform import <resource address> <virtual machine id>`) or delete it before re-applying.",
			action, cause, activityID, name,
		)
	}

	vm, readErr := read(ctx, candidate)
	if readErr == nil && vm == nil {
		// One not-found is not proof; see confirmVMAbsence.
		vm, readErr = confirmVMAbsence(ctx, read, candidate)
	}
	switch {
	case readErr == nil && vm == nil:
		// Absence confirmed by repeated read-backs. Nothing exists to track, and
		// recording the id would poison every later refresh (see case 4).
		//
		// The operator is still told to AUDIT before re-applying, like every other
		// branch that records nothing. Telling them to re-apply outright would be
		// the one instruction that can manufacture the incident this code exists to
		// prevent: if the reads crossed an indexing window after all, the virtual
		// machine materialises unattended and the re-apply creates a SECOND one.
		return diag.Errorf(
			"%s: %s. Activity %s referenced virtual machine %s, but %d read-backs found no such virtual machine — the platform appears to have rolled the creation back, and NOTHING was recorded in the Terraform state. Confirm no virtual machine named %q exists before re-applying; if one does, it is ORPHANED — import it (`terraform import <resource address> <virtual machine id>`) or delete it first.",
			action, cause, activityID, candidate, vmAbsenceAttempts+1, name,
		)

	case readErr != nil:
		// Inconclusive: never let a read that proves nothing orphan a virtual
		// machine that probably exists.
		d.SetId(candidate)
		return diag.Errorf(
			"%s: %s. Activity %s created virtual machine %s, whose read-back was inconclusive (%s). It was recorded in the Terraform state (tainted) so it is NOT orphaned: run `terraform destroy` to remove it, or `terraform apply` to replace it. If you verify the virtual machine is actually complete, `terraform untaint` it and re-apply to finish the configuration instead of recreating it.",
			action, cause, activityID, candidate, readErr,
		)

	case vm.Template:
		// A template is never an object this resource created. Adopting one
		// would taint it, and the next apply would DESTROY a template other
		// virtual machines are deployed from.
		return diag.Errorf(
			"%s: %s. Activity %s designated %s, which is a TEMPLATE, not the virtual machine this resource creates; it was deliberately NOT recorded in the Terraform state. If a virtual machine named %q was created it is ORPHANED — audit it and either import it (`terraform import <resource address> <virtual machine id>`) or delete it before re-applying.",
			action, cause, activityID, candidate, name,
		)

	case vm.Name != name:
		// The created virtual machine carries the name that was requested
		// (measured against the API: no normalisation, no suffix). A different
		// name means the activity designated some other object, so adoption
		// fails closed — recording nothing and naming the candidate is far
		// better than tainting a virtual machine this resource never created.
		return diag.Errorf(
			"%s: %s. Activity %s designated virtual machine %s, but it is named %q while %q was requested; it was deliberately NOT recorded in the Terraform state. Audit the virtual machine named %q: if it was created, import it (`terraform import <resource address> <virtual machine id>`) or delete it before re-applying.",
			action, cause, activityID, candidate, vm.Name, name, name,
		)

	default:
		d.SetId(candidate)
		// machine_manager_id is what computeVirtualMachineRead needs to confirm
		// or refuse a deletion when the per-id read comes back empty. Recording
		// it here keeps a later refresh able to resolve this resource.
		var setErr error
		if vm.MachineManager.ID != "" {
			setErr = d.Set("machine_manager_id", vm.MachineManager.ID)
		}
		msg := fmt.Sprintf(
			"%s: %s. Virtual machine %s was created and HAS been recorded in the Terraform state (tainted), so it is not orphaned: run `terraform destroy` to remove it, or `terraform apply` to replace it. If you verify the virtual machine is actually complete, `terraform untaint` it and re-apply to finish the configuration instead of recreating it.",
			action, cause, candidate,
		)
		if setErr != nil {
			msg += fmt.Sprintf(" Its machine manager id could not be written to the state (%s); a later refresh may need `terraform import` to resolve it.", setErr)
		}
		return diag.Errorf("%s", msg)
	}
}
