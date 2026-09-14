package provider

import (
	"context"
	"fmt"

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
// before `state.completed.result` carries that same id. That is the
// authoritative correlation used here — never a lookup by name, which could
// match a homonym the provider did not create.
//
// Adopting an id is a STATE-CRITICAL decision: the adopted resource is
// persisted as TAINTED, so the next apply DESTROYS whatever it points at. The
// evidence required is therefore strict, and every ambiguous case fails closed
// with a diagnostic that tells the operator exactly what to audit.

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
			"%s: %s. The virtual machine %s IS recorded in the Terraform state (tainted): run `terraform destroy` to remove it, or `terraform apply` to replace it — no manual cleanup is needed.",
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
	switch {
	case readErr == nil && vm == nil:
		// Definitive absence: the platform rolled the creation back. Nothing
		// exists to track, and recording the id would poison every later refresh.
		return diag.Errorf(
			"%s: %s. Activity %s referenced virtual machine %s, but it no longer exists — the platform rolled the creation back. Nothing was recorded in the Terraform state; re-run `terraform apply`.",
			action, cause, activityID, candidate,
		)

	case readErr != nil:
		// Inconclusive: never let a read that proves nothing orphan a virtual
		// machine that probably exists.
		d.SetId(candidate)
		return diag.Errorf(
			"%s: %s. Activity %s created virtual machine %s, whose read-back was inconclusive (%s). It was recorded in the Terraform state (tainted) so it is NOT orphaned: run `terraform destroy` to remove it, or `terraform apply` to replace it.",
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
			"%s: %s. Virtual machine %s was created and HAS been recorded in the Terraform state (tainted), so it is not orphaned: run `terraform destroy` to remove it, or `terraform apply` to replace it.",
			action, cause, candidate,
		)
		if setErr != nil {
			msg += fmt.Sprintf(" Its machine manager id could not be written to the state (%s); a later refresh may need `terraform import` to resolve it.", setErr)
		}
		return diag.Errorf("%s", msg)
	}
}
