package provider

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const cloudInitRecovery = "The VM may have partial or duplicate vApp properties. Keep it stopped. Do not reapply or power it on until all backend activities/tasks have finished and the properties have been reconciled. Terraform retains the previous cloud_init intent; this is not a backend rollback."

// cloudInitStoppedOnly is shown when the virtual machine is running and the
// operator has not authorised a power cycle. The sequence below is a non-atomic
// add-then-delete that transiently leaves two properties with the same textual
// ID, so it must not be observed by a booting guest.
const cloudInitStoppedOnly = "cloud_init can only be changed while the virtual machine is stopped: the public vApp API has no atomic edit, so the update transiently leaves duplicate property IDs. Set `allow_vm_restart = true` to let the provider power the virtual machine off and back on around the change, or set `power_state = \"off\"` to apply it and leave the machine stopped."

type cloudInitEdit struct {
	original    map[string]interface{}
	replacement map[string]interface{}
}

// cloudInitRun carries the bookkeeping that decides, on failure, whether the
// virtual machine may be powered back on.
//
// The rule is deliberately strict: a power-on is allowed ONLY when no vApp write
// was ever ATTEMPTED. Once a write has left the provider its outcome can be
// unknown — a lost response still mutates the vApp — and booting a machine whose
// properties may hold a duplicate or a half-applied edit is precisely what
// cloudInitRecovery instructs the operator not to do. A provider that printed
// that instruction and then powered the VM on anyway would be contradicting its
// own diagnostic. This is the one place where this path deliberately does NOT
// mirror the sizing power-cycle (#397): a sizing PATCH is a single atomic call,
// so compensating it by powering back on is safe; this is a multi-step
// transaction and it is not.
type cloudInitRun struct {
	targetOn   bool // declared final power intent, captured before any read
	poweredOff bool // the shutdown was issued AND confirmed by a live snapshot
	mutated    bool // at least one vApp write was attempted; outcome may be unknown
	verified   bool // the whole vApp transaction completed and was verified
	// observed is the last power state a LIVE read confirmed, in the schema's
	// vocabulary ("on"/"off"), empty until one does. A completed activity never
	// sets it: the platform can accept a request whose response is lost.
	observed string
	// proven is the last vApp this invocation OBSERVED and accepted. It is not by
	// itself a certificate of soundness: the shutdown stores it before the
	// re-plan runs, so it can already carry a duplicate another writer left
	// behind. That is why cloudInitSafeToStart checks the snapshot AND the
	// duplicates AND the power state, rather than trusting this field alone.
	proven *client.VAppSnapshot

	// Planner inputs, carried so the post-shutdown re-plan uses exactly the
	// same inputs as the pre-shutdown feasibility plan.
	oldMap, newMap, deploy map[string]interface{}
}

// A separate update, performed while the VM is stopped, avoids changing creation
// semantics or running hardware/power handlers while duplicate property IDs
// temporarily exist. A running VM is powered off first when the operator opted in
// with allow_vm_restart, or asked for power_state = "off" anyway.
func updateVMwareCloudInit(ctx context.Context, d *schema.ResourceData, c *client.Client) (diags diag.Diagnostics) {
	old, newValue := d.GetChange("cloud_init")
	id := d.Id()

	// Capture the power intent BEFORE anything can repopulate ResourceData:
	// computeVirtualMachineRead writes power_state from the LIVE machine, so
	// reading it afterwards would return what was observed, not what was declared.
	run := &cloudInitRun{targetOn: d.Get("power_state").(string) == "on"}
	allowRestart := d.Get("allow_vm_restart").(bool)

	defer func() {
		if !diags.HasError() {
			return
		}
		d.SetId(id)
		// A VERIFIED transaction must not be rolled back. Once the vApp holds the
		// new values, restoring the old intent would make every later apply refuse
		// — an added key then looks unadopted, a removed one looks lost — and
		// the refresh cannot repair it: it never adds a key back, and it never
		// removes one it can no longer observe.
		// Only a transaction that did not complete gives back the prior intent.
		if run.verified {
			return
		}
		if err := d.Set("cloud_init", old); err != nil {
			diags = append(diags, diag.Errorf("could not restore previous cloud_init intent")...)
		}
	}()

	// The SDK persists PLANNED values when an update returns an error, so a
	// machine this apply deliberately left stopped would be recorded as running
	// — the opposite of what happened, and a contradiction of this path's own
	// rule that a power action is only ever recorded after a live read confirms
	// its effect. Record what was actually observed instead.
	//
	// Only positive evidence counts: run.observed is set solely by a live read
	// that returned a stable state. With none, the prior state stands, which is
	// what the callers that freeze the state with d.Partial already arrange.
	defer func() {
		if !diags.HasError() || run.observed == "" {
			return
		}
		if err := d.Set("power_state", run.observed); err != nil {
			diags = append(diags, diag.Errorf("could not record the observed power state")...)
		}
	}()

	// The AUTHORITATIVE copy of the sole-change rule. The plan-time guard reports
	// it earlier, but fails open on a value Terraform has not resolved yet, so
	// this one still decides.
	if name := cloudInitOtherChangedAttribute(d); name != "" {
		return refuseBeforeAnyMutation(d, diag.Errorf("cannot change %q in the same apply as cloud_init: apply the cloud_init change on its own", name))
	}

	run.deploy = d.Get("deploy_options").(map[string]interface{})
	run.oldMap, run.newMap = old.(map[string]interface{}), newValue.(map[string]interface{})

	// Feasibility BEFORE downtime. A configuration that could never be applied —
	// an ownership collision, drift, a value the API would normalize, a disabled
	// vApp — is refused here, while the machine is still running. Powering a
	// production VM off only to discover the plan was impossible is not acceptable.
	before, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
	if err != nil {
		return refuseBeforeAnyMutation(d, diag.Errorf("cannot obtain a complete vApp snapshot; cloud_init was not changed"))
	}
	planned, err := planCloudInitEdits(before, run.oldMap, run.newMap, run.deploy)
	if err != nil {
		return refuseBeforeAnyMutation(d, diag.Errorf("cannot update cloud_init: %s", err))
	}

	// Nothing to write means nothing to protect: a no-op must never cost an
	// interruption, and the stopped-only verifications below would be meaningless
	// (and would refuse a running machine) with an empty plan.
	if len(planned) > 0 {
		if powerDiags := cloudInitPrepare(ctx, d, c, id, run, allowRestart, &before, planned); powerDiags != nil {
			return powerDiags
		}
		if writeDiags := cloudInitApplyEdits(ctx, d, c, id, run, before, planned); writeDiags != nil {
			return writeDiags
		}
	}
	if len(planned) == 0 && before.PowerState == "stopped" {
		run.proven = before
	}
	// The vApp now provably holds the declared values — either they were written
	// and verified, or there was nothing to write because it already held them.
	// This is the transaction boundary, and it is deliberately BEFORE the general
	// read: that read fetches tags and backup policies, and a failure there must
	// not roll back an intent the machine has already accepted.
	run.verified = true
	// General read still maps the established schema; it does not hydrate or
	// rewrite cloud_init for existing resources.
	if diags = computeVirtualMachineRead(ctx, d, c); diags.HasError() {
		return cloudInitFail(ctx, c, d, id, run, "final VM read", "")
	}
	// The read above recorded the OBSERVED power state. Settle the DECLARED one
	// last, in BOTH directions, so the state describes the machine the operator
	// asked for — including a machine that was already stopped and must end up
	// running, or one left running by a no-op and configured to be off.
	return append(diags, cloudInitSettlePower(ctx, c, d, id, run)...)
}

// cloudInitPrepare brings the machine to a stopped state the sequence can be
// written against, and re-plans once it is there.
//
// Only "running" and "stopped" are evidence. Anything else — suspended, a
// transitional state, an empty field — is NOT proof that no shutdown is needed,
// so it fails closed instead of being read as "not running".
func cloudInitPrepare(ctx context.Context, d *schema.ResourceData, c *client.Client, id string, run *cloudInitRun, allowRestart bool, before **client.VAppSnapshot, planned []cloudInitEdit) diag.Diagnostics {
	switch (*before).PowerState {
	case "stopped":
		// This state came from a LIVE read, so it is an observation, and the
		// update can still fail after this point with nothing of its own to
		// record — a failed write leaves the machine exactly as stopped as it
		// found it. Without recording it here the SDK persists the PLANNED power
		// state on such a failure, so a machine nobody ever powered off is
		// written down as running: the one state it is known not to be in.
		cloudInitRecordPower(run, "stopped")
		return nil
	case "running":
		if run.targetOn && !allowRestart {
			return refuseBeforeAnyMutation(d, diag.Errorf("%s", cloudInitStoppedOnly))
		}
		stopped, shutdownDiags := cloudInitShutdown(ctx, d, c, id, run)
		if shutdownDiags != nil {
			return shutdownDiags
		}
		// Re-plan against the machine that will actually be mutated. The
		// pre-shutdown plan proved feasibility; it is not the plan to apply.
		settled, err := planCloudInitEdits(stopped, run.oldMap, run.newMap, run.deploy)
		if err != nil {
			return cloudInitAbort(ctx, c, d, id, run, diag.Errorf("cannot update cloud_init: %s", err))
		}
		// A plan that changed across the shutdown means the descriptors this
		// update replays are no longer the ones it planned against. Fail closed
		// rather than write a stale plan.
		if !reflect.DeepEqual(settled, planned) {
			return cloudInitAbort(ctx, c, d, id, run, diag.Errorf("the vApp changed while the virtual machine was being powered off; cloud_init was not changed"))
		}
		*before = stopped
		return nil
	default:
		// The value itself is NOT echoed: ReadVAppSnapshot does not constrain this
		// field, so an arbitrary backend string must not reach a diagnostic.
		return refuseBeforeAnyMutation(d, diag.Errorf("the virtual machine is in an unsupported power state; cloud_init can only be changed from a machine that is known to be running or stopped"))
	}
}

// cloudInitApplyEdits performs the non-atomic add-then-delete sequence against a
// machine already proven stopped, verifying a full snapshot at every step.
func cloudInitApplyEdits(ctx context.Context, d *schema.ResourceData, c *client.Client, id string, run *cloudInitRun, before *client.VAppSnapshot, planned []cloudInitEdit) diag.Diagnostics {
	expected := before
	for _, edit := range planned {
		if err := verifyVAppSnapshot(ctx, c, id, expected); err != nil {
			return cloudInitFail(ctx, c, d, id, run, "pre-write verification", "")
		}
		if edit.replacement != nil {
			// From here the vApp may have changed even if the call reports failure.
			run.mutated = true
			activity, err := c.Compute().VirtualMachine().AddVAppProperty(ctx, id, edit.replacement)
			if err != nil {
				return cloudInitFail(ctx, c, d, id, run, "adding property", activity)
			}
			// A silent waiter keeps backend-echoed cloud-init out of the DIAGNOSTICS
			// and out of the activity logs. It does NOT protect the request body:
			// that is masked by path in the logging transport, because a descriptor
			// carries its secret as a VALUE ("id":"password") that no key-based
			// redaction can recognise.
			if _, err = c.Activity().WaitForCompletion(ctx, activity, nil); err != nil {
				return cloudInitFail(ctx, c, d, id, run, "waiting for property addition", activity)
			}
			actual, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
			if err != nil {
				return cloudInitFail(ctx, c, d, id, run, "reading added property", activity)
			}
			if err = verifyVAppAddition(expected, actual, edit.replacement); err != nil {
				return cloudInitFail(ctx, c, d, id, run, "verifying added property", activity)
			}
			expected = actual
		}
		if edit.original != nil {
			if err := verifyVAppSnapshot(ctx, c, id, expected); err != nil {
				return cloudInitFail(ctx, c, d, id, run, "pre-delete verification", "")
			}
			key := int(edit.original["key"].(float64))
			run.mutated = true
			activity, err := c.Compute().VirtualMachine().DeleteVAppProperty(ctx, id, key)
			if err != nil {
				return cloudInitFail(ctx, c, d, id, run, "deleting original property", activity)
			}
			// A silent waiter keeps backend-echoed cloud-init out of the DIAGNOSTICS
			// and out of the activity logs. It does NOT protect the request body:
			// that is masked by path in the logging transport, because a descriptor
			// carries its secret as a VALUE ("id":"password") that no key-based
			// redaction can recognise.
			if _, err = c.Activity().WaitForCompletion(ctx, activity, nil); err != nil {
				return cloudInitFail(ctx, c, d, id, run, "waiting for original property deletion", activity)
			}
			remaining := make([]map[string]interface{}, 0, len(expected.Properties)-1)
			for _, p := range expected.Properties {
				if p["key"] != float64(key) {
					remaining = append(remaining, p)
				}
			}
			expected = &client.VAppSnapshot{ID: id, PowerState: "stopped", Enabled: true, Properties: remaining}
			if err := verifyVAppSnapshot(ctx, c, id, expected); err != nil {
				return cloudInitFail(ctx, c, d, id, run, "verifying original property deletion", activity)
			}
		}
	}

	// Prove the transaction is complete while the machine is still stopped. Every
	// snapshot comparison asserts "stopped", so this has to happen before any
	// power-on — and that ordering is itself a check that nobody started the VM
	// mid-sequence.
	if err := verifyVAppSnapshot(ctx, c, id, expected); err != nil {
		return cloudInitFail(ctx, c, d, id, run, "final vApp verification", "")
	}
	run.proven = expected
	return nil
}

// cloudInitSettlePower drives the virtual machine to the declared final power
// state after a successful update. It is NOT gated on run.poweredOff: a machine
// that was already stopped when the apply started must still be started when the
// configuration says power_state = "on". Starting a stopped machine interrupts
// nothing, so it needs no allow_vm_restart opt-in — that flag authorises
// INTERRUPTING a running machine, which is a different question.
func cloudInitSettlePower(ctx context.Context, c *client.Client, d *schema.ResourceData, id string, run *cloudInitRun) diag.Diagnostics {
	want, action, recorded := "stopped", "off", "off"
	if run.targetOn {
		want, action, recorded = "running", "on", "on"
	}
	observed, err := cloudInitLivePower(ctx, c, id)
	cloudInitRecordPower(run, observed)
	if err != nil {
		return diag.Errorf("cloud_init was applied and verified but the virtual machine's power state could not be read afterwards; re-run apply to converge. Terraform's recorded power state may be stale until the next refresh")
	}
	if observed != want {
		// "not the target" is not the same as "the opposite of the target". A
		// transitional or unrecognised state is not evidence that acting is safe:
		// powering on could race a shutdown that has not settled.
		if observed == "" {
			return diag.Errorf("cloud_init was applied and verified but the virtual machine is in an unsupported power state, so its power state was left untouched; settle it manually or re-run apply")
		}
		if run.targetOn {
			// Same rule as compensation: never boot a guest into a vApp this
			// invocation can see is no longer the one it verified.
			if err := cloudInitSafeToStart(ctx, c, id, run); err != nil {
				return diag.Errorf("cloud_init was applied and verified, but the virtual machine was deliberately left powered off because %s; reconcile the vApp, then start it yourself or re-run apply", err)
			}
		}
		vm, derr := cloudInitReadVM(ctx, c, id, "settle its power state after the cloud_init update")
		if derr != nil {
			return derr
		}
		powerErr := cloudInitPowerAction(ctx, c, vm, id, action)
		// Whether the call failed or the activity completed, neither settles the
		// question: the platform can accept a request whose response is lost. Only
		// a live read decides, and an unreadable one is reported as unknown.
		observed, err = cloudInitLivePower(ctx, c, id)
		cloudInitRecordPower(run, observed)
		if err != nil || observed != want {
			if powerErr != nil {
				return diag.Errorf("cloud_init was applied and verified but the virtual machine could not be confirmed powered %s afterwards (%s); check its power state, then set it manually or re-run apply", action, powerErr)
			}
			return diag.Errorf("cloud_init was applied and verified but the virtual machine could not be confirmed powered %s afterwards; check its power state, then re-run apply to converge", action)
		}
	}
	if err := d.Set("power_state", recorded); err != nil {
		return diag.Errorf("could not record the final power state")
	}
	return nil
}

// cloudInitLivePower returns "running", "stopped", or "" when the live state is
// anything else. Collapsing every other value to "" is deliberate: an
// unrecognised or transitional state must never be read as evidence of the
// opposite stable state, and the value itself is never echoed because
// ReadVAppSnapshot does not constrain that field.
func cloudInitLivePower(ctx context.Context, c *client.Client, id string) (string, error) {
	live, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
	if err != nil {
		return "", err
	}
	switch live.PowerState {
	case "running", "stopped":
		return live.PowerState, nil
	}
	return "", nil
}

// cloudInitSafeToStart reports whether this invocation may power the machine ON.
//
// "I wrote nothing" is NOT the same as "the machine is safe to start". Another
// writer can have left a duplicate property behind in the meantime, and a
// transitional power state is not a stopped one. Powering a machine OFF never
// needs this check — stopping a guest cannot boot it into a half-written vApp —
// so only the power-on paths call it, and both refuse rather than guess.
func cloudInitSafeToStart(ctx context.Context, c *client.Client, id string, run *cloudInitRun) error {
	live, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
	if err != nil {
		return fmt.Errorf("its vApp could not be read")
	}
	// This read is a live observation, and on every path that reaches here the
	// caller is about to record a power state on an error. Hand it this one
	// rather than let an older one stand: the machine may have been started by
	// another actor while it was down, in which case the provider's own earlier
	// "off" is stale and would be recorded over a machine that is running. An
	// unrecognised or transitional state still records nothing.
	cloudInitRecordPower(run, live.PowerState)
	if live.PowerState != "stopped" {
		return fmt.Errorf("it is not observably stopped")
	}
	// Duplicate TEXTUAL ids are the hazard this whole update exists to avoid,
	// and ReadVAppSnapshot does not reject them: it only enforces unique numeric
	// keys. Only the keys this apply manages are considered, so a machine with
	// unrelated legacy duplicates is not held hostage by them.
	managed := map[string]bool{}
	for k := range run.oldMap {
		managed[k] = true
	}
	for k := range run.newMap {
		managed[k] = true
	}
	seen := map[string]bool{}
	for _, p := range live.Properties {
		propertyID, _ := p["id"].(string)
		if !managed[propertyID] {
			continue
		}
		if seen[propertyID] {
			return fmt.Errorf("its vApp holds duplicate entries for a managed cloud_init key")
		}
		seen[propertyID] = true
	}
	if run.proven == nil || !sameVAppSnapshot(run.proven, live) {
		return fmt.Errorf("its vApp no longer matches the state this apply verified")
	}
	return nil
}

// cloudInitShutdown powers the machine off before the vApp sequence.
//
// A power request can be ACCEPTED by the platform even when the call reports an
// error, so a failure here is reported as an UNCERTAIN outcome rather than as
// "nothing happened", and is never followed by an immediate power-on — which
// could race a shutdown that is still completing.
func cloudInitShutdown(ctx context.Context, d *schema.ResourceData, c *client.Client, id string, run *cloudInitRun) (*client.VAppSnapshot, diag.Diagnostics) {
	vm, derr := cloudInitReadVM(ctx, c, id, "power the virtual machine off")
	if derr != nil {
		return nil, refuseBeforeAnyMutation(d, derr)
	}
	if err := cloudInitPowerAction(ctx, c, vm, id, "off"); err != nil {
		return nil, refuseBeforeAnyMutation(d, diag.Errorf("the virtual machine could not be confirmed powered off for a cloud_init update (%s); no vApp property was written and cloud_init was not changed. The power-off may still be in progress: check the virtual machine before re-running apply", err))
	}
	// A completed activity is not evidence of the resulting state. An UNCONFIRMED
	// shutdown must never be "compensated" by a power-on, which could race a
	// shutdown still settling — so both failures below refuse outright rather
	// than routing through cloudInitAbort, and run.poweredOff (which is what
	// authorises a compensating power-on) is set only once a live snapshot says
	// stopped. The two together are deliberate belt and braces.
	stopped, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
	if err != nil {
		return nil, refuseBeforeAnyMutation(d, diag.Errorf("the virtual machine could not be re-read after the power-off; no vApp property was written and cloud_init was not changed. Check its power state before re-running apply"))
	}
	if stopped.PowerState != "stopped" {
		return nil, refuseBeforeAnyMutation(d, diag.Errorf("the virtual machine is not stopped after the power-off reported completion; no vApp property was written and cloud_init was not changed. The shutdown may still be settling: check the machine before re-running apply"))
	}
	run.poweredOff = true
	run.observed = "off"
	run.proven = stopped
	return stopped, nil
}

// cloudInitReadVM reads the virtual machine for the datacenter id a power request
// needs. The general read tolerates HTTP 206 and does not check the identity it
// got back, so the result is validated here before anything powers a machine on
// the strength of it.
func cloudInitReadVM(ctx context.Context, c *client.Client, id, purpose string) (*client.VirtualMachine, diag.Diagnostics) {
	vm, err := c.Compute().VirtualMachine().Read(ctx, id)
	if err != nil || vm == nil || vm.ID != id || vm.Datacenter.ID == "" {
		return nil, diag.Errorf("cannot read the virtual machine in order to %s", purpose)
	}
	return vm, nil
}

// cloudInitPowerAction is this path's own power helper. It deliberately does NOT
// reuse vmwarePowerAction: that one interpolates raw API errors into diagnostics
// and waits with getWaiterOptions, which logs backend messages at debug level.
// Everything on the cloud_init path keeps backend text out of diagnostics and
// out of the ACTIVITY logs, because a cloud-init payload can hold credentials
// and an activity error can carry a backend description, result or reason.
//
// This says nothing about request and response BODIES, which the logging
// transport handles: see the vapp_properties clause in its RoundTrip.
func cloudInitPowerAction(ctx context.Context, c *client.Client, vm *client.VirtualMachine, id, action string) error {
	req := &client.PowerRequest{ID: id, DatacenterId: vm.Datacenter.ID, PowerAction: action}
	if action == "on" {
		reco, err := helpers.GetPowerRecommendation(vm, "on", ctx, c)
		if err != nil {
			return fmt.Errorf("no placement recommendation could be obtained")
		}
		req.Recommendation = reco
	}
	activity, err := c.Compute().VirtualMachine().Power(ctx, req)
	if err != nil {
		return fmt.Errorf("the power %s request could not be confirmed as sent", action)
	}
	// Silent waiter, for the same reason as the vApp writes above.
	if _, err := c.Activity().WaitForCompletion(ctx, activity, nil); err != nil {
		return fmt.Errorf("the power %s activity did not complete", action)
	}
	return nil
}

// cloudInitFail ends a failed update.
//
// The machine is powered back on ONLY when no vApp write was ever attempted. Once
// one has, the outcome can be unknown and cloudInitRecovery tells the operator to
// keep the machine stopped until the properties have been reconciled; honouring
// that in code is the point of run.mutated.
func cloudInitFail(ctx context.Context, c *client.Client, d *schema.ResourceData, id string, run *cloudInitRun, stage, activity string) diag.Diagnostics {
	if run.verified {
		// The transaction completed and was verified; what failed came after it.
		// The declared values ARE on the machine and Terraform keeps them, so the
		// recovery procedure for an UNCERTAIN transaction does not apply here and
		// must not be printed: it would send the operator to reconcile vApp
		// properties that are already correct.
		return diag.Errorf("cloud_init was applied and verified, but %s failed afterwards. The vApp properties hold the declared values and Terraform has recorded them; re-run apply to converge the rest of the state", stage)
	}
	if run.mutated {
		return cloudInitUpdateError(stage, activity)
	}
	return cloudInitAbort(ctx, c, d, id, run,
		diag.Errorf("cloud_init update refused during %s; no vApp property was written and cloud_init was not changed", stage))
}

// cloudInitAbort ends an update that provably wrote nothing to the vApp. It gives
// back the power state the provider itself took away, then returns the ORIGINAL
// diagnostic — never the recovery outcome — so the apply still fails loudly.
func cloudInitAbort(ctx context.Context, c *client.Client, d *schema.ResourceData, id string, run *cloudInitRun, cause diag.Diagnostics) diag.Diagnostics {
	// Defence in depth. No current caller reaches this with a write attempted,
	// and none ever should: this function powers the machine on. Enforcing it
	// here means a future caller cannot reintroduce the hazard by accident.
	if run.mutated {
		return cloudInitUpdateError("an unexpected failure after a vApp write", "")
	}
	if !run.poweredOff || !run.targetOn {
		return refuseBeforeAnyMutation(d, cause)
	}
	// Give the machine back only on positive evidence that starting it is safe.
	// The very reason this apply is aborting may BE that evidence: a re-plan that
	// rejected a duplicate left by another writer has told us, in so many words,
	// that the guest must not boot now.
	if err := cloudInitSafeToStart(ctx, c, id, run); err != nil {
		return cloudInitRefuse(d, run, append(cause, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "virtual machine was deliberately left powered off",
			Detail:   fmt.Sprintf("The provider powered the virtual machine off for a cloud_init update and wrote nothing, but did not start it again because %s. Reconcile the vApp, then power it on yourself or re-run `terraform apply`.", err),
		}))
	}
	if vm, derr := cloudInitReadVM(ctx, c, id, "power the virtual machine back on"); derr == nil {
		_ = cloudInitPowerAction(ctx, c, vm, id, "on")
	}
	// Neither a completed activity nor a failed call settles the question: the
	// platform can accept a request whose response is lost. Report only what a
	// live snapshot shows, and say so plainly when it cannot be read.
	live, _ := cloudInitLivePower(ctx, c, id)
	cloudInitRecordPower(run, live)
	if live == "running" {
		return cloudInitRefuse(d, run, append(cause, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "virtual machine was powered back on",
			Detail:   "The provider powered the virtual machine off for a cloud_init update, wrote nothing to the vApp, and has confirmed it running again. Re-run `terraform apply` to converge.",
		}))
	}
	left := append(cause, diag.Diagnostic{
		Severity: diag.Error,
		Summary:  "virtual machine could not be confirmed running again",
		Detail:   "The provider powered the virtual machine off for a cloud_init update and could not confirm it running again. No vApp property was written, so the machine is safe to start: check its power state and start it manually, or re-run `terraform apply`.",
	})
	return cloudInitRefuse(d, run, left)
}

func cloudInitUpdateError(stage, activity string) diag.Diagnostics {
	// Never echo API errors, property values or descriptors: cloud-init can contain
	// credentials, and API error messages may include the rejected request body.
	// The activity id is worth printing for support, but it is taken verbatim from
	// a Location header the client does not constrain, so it is printed only when
	// it looks like an identifier and is otherwise reported as unavailable.
	return diag.Errorf("cloud_init update failed during %s (activity %s). %s", stage, safeActivityRef(activity), cloudInitRecovery)
}

// safeActivityRef renders an activity id only if it cannot carry arbitrary text.
func safeActivityRef(activity string) string {
	if activity == "" {
		return "none issued"
	}
	if len(activity) > 64 {
		return "not reportable"
	}
	for _, r := range activity {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return "not reportable"
		}
	}
	return strconv.Quote(activity)
}

// planCloudInitEdits is a pure planner: it decides WHAT to change and refuses
// anything it cannot replay faithfully. It no longer refuses a running machine —
// the caller owns power, and runs this once before any shutdown precisely so an
// impossible configuration costs no downtime. The stopped-only invariant is
// still enforced on every write, by sameVAppSnapshot and verifyVAppAddition.
func planCloudInitEdits(vm *client.VAppSnapshot, old, desired, deploy map[string]interface{}) ([]cloudInitEdit, error) {
	if !vm.Enabled {
		return nil, fmt.Errorf("the vApp is not enabled on this virtual machine; cloud_init cannot be changed")
	}
	keys := map[string]bool{}
	for k, v := range old {
		if want, ok := desired[k]; !ok || want != v {
			keys[k] = true
		}
	}
	for k, v := range desired {
		if was, ok := old[k]; !ok || was != v {
			keys[k] = true
		}
	}
	ordered := make([]string, 0, len(keys))
	for k := range keys {
		ordered = append(ordered, k)
	}
	sort.Strings(ordered)
	edits := []cloudInitEdit{}
	for _, id := range ordered {
		if !transparentVAppString(id) || utf8.RuneCountInString(id) > 60 {
			return nil, fmt.Errorf("a changed property ID cannot be transmitted without API normalization")
		}
		if _, ok := deploy[id]; ok {
			if _, present := desired[id]; !present {
				// deploy_options owns this key's live value, so dropping it from
				// cloud_init means "stop recording it here", not "delete it from
				// the machine": deleting would destroy a value the configuration
				// still declares. Skipping is also the only non-destructive way
				// out of a collision, since deploy_options is ForceNew and
				// editing it replaces the machine.
				continue
			}
			return nil, fmt.Errorf("a changed cloud_init key also belongs to deploy_options; resolve ownership first")
		}
		var live map[string]interface{}
		for _, p := range vm.Properties {
			if p["id"] == id {
				if live != nil {
					return nil, fmt.Errorf("duplicate changed vApp property IDs require manual reconciliation")
				}
				live = p
			}
		}
		was, managed := old[id]
		want, present := desired[id]
		if managed && live == nil {
			// A property this resource recorded is no longer on the machine:
			// deleted in the console, or by another tool.
			//
			// Dropping the key from the configuration asks for an end state the
			// machine is ALREADY in, and asks for no write — there is nothing
			// left to delete. Refusing it was a dead end with no exit but
			// `terraform state rm` followed by `terraform import`, since the
			// value could not be changed either (just below) and so NO
			// configuration converged. Converging here also leaves the state
			// and the machine agreeing, which is strictly better than the
			// divergence a still-declared key leaves behind.
			if !present {
				continue
			}
			// Writing a new value is a different matter: the replacement below
			// clones the LIVE descriptor, and there is none. A fresh property
			// would carry invented metadata and would silently resurrect
			// something a third party deliberately deleted.
			return nil, fmt.Errorf("a previously managed property is missing; reconcile it before applying (metadata cannot be reconstructed)")
		}
		// A key the configuration declares is a key the configuration owns, even
		// when the machine already carries it. Terraform is the source of truth,
		// so the declared value wins and overwrites the live one.
		//
		// Refusing to adopt used to look like the safe answer, and for a machine
		// Terraform created it costs nothing — a key it created is always
		// managed. For an IMPORTED machine it is a dead end: an import records no
		// cloud_init, so every declared key is unmanaged and every one of them
		// already exists on the machine by construction. No configuration could
		// converge, not even one declaring the exact values the machine held.
		//
		// Adoption is safe to express here because the write is not a fresh
		// property: the replacement below clones the LIVE descriptor and swaps
		// only its value, so OVF-provided metadata survives, and vAppPayload
		// still refuses anything the public API cannot replay faithfully.
		//
		// What adoption does carry is a lifecycle: an adopted key is managed from
		// then on, so removing it from the configuration later DELETES the vApp
		// property, exactly as for a key Terraform created.
		//
		// Drift is only meaningful against a recorded intent. An unmanaged key
		// has none — there is nothing it could have drifted from — so the check
		// below is confined to keys this resource already tracks.
		//
		// A value the platform DEFORMED on the way in is not drift. It is this
		// provider's own recorded intent, in the form the platform chose to
		// store it.
		//
		// Reporting it as drift made the upgrade to 1.13.0 a dead end for every
		// machine created before it whose cloud_init carried a '/' — most RSA
		// public keys, every seedfrom URL. State holds the declared form, the
		// vApp holds the escaped one, and both remedies the upgrade note offers
		// land here: removing the key is a change, and so is replacing it with a
		// transmittable value. The vApp cannot be "reconciled" to the recorded
		// value either, since the platform escapes it again on the way in. The
		// only way out was `terraform state rm` followed by `terraform import`.
		//
		// The recognition works by REVERSING the escaping (see
		// cloudInitPlatformUnescape), not by reproducing it: the two write paths
		// were measured escaping different subsets of the same eight characters,
		// so there is no single forward form to compare against. A transform
		// outside the measured table is left alone and still reports drift,
		// which is why the guide keeps the state surgery as the exit for those.
		if managed && live != nil && live["value"] != was && (!present || live["value"] != want) &&
			!cloudInitIsPlatformEscapeOf(live["value"], was) {
			return nil, fmt.Errorf("a changed vApp property has drifted; reconcile it before applying")
		}
		if present && live != nil && live["value"] == want {
			continue
		}
		edit := cloudInitEdit{original: live}
		if present {
			value, ok := want.(string)
			if !ok || !transparentVAppString(value) {
				return nil, fmt.Errorf("a changed cloud_init value is empty or would be trimmed/HTML-escaped by the API")
			}
			var payload map[string]interface{}
			if live == nil {
				payload = map[string]interface{}{"id": id, "label": id, "type": "string", "userConfigurable": true}
			} else {
				var err error
				// The old value is not replayed. Validate the desired value and
				// preserved metadata, including legacy values only when retained.
				clone := make(map[string]interface{}, len(live))
				for k, v := range live {
					clone[k] = v
				}
				clone["value"] = value
				payload, err = vAppPayload(clone)
				if err != nil {
					return nil, err
				}
			}
			payload["value"] = value
			edit.replacement = payload
		}
		edits = append(edits, edit)
	}
	return edits, nil
}

// The public API trims and HTML-escapes all strings, including '/'. Reject a
// non-round-trippable request before any write instead of persisting false intent.
// cloudInitKeys is the documented set of cloud-init property names.
var cloudInitKeys = regexp.MustCompile(strings.Join([]string{
	"^user-data$",
	"^network-config$",
	"^public-keys$",
	"^instance-id$",
	"^password$",
	"^hostname$",
	"^seedfrom$"}, "|"))

// validateCloudInitMap refuses a key the platform does not know, and a VALUE it
// cannot store faithfully.
//
// The value rule is the one the update path already applies; enforcing it at
// CREATION too closes a silent failure. The deployment API was measured to
// mutate what it is given: it trims surrounding whitespace, and it HTML-escapes
// '/' into the six literal characters &#x2F;. That escaped form is what reaches
// the machine — observed in vCenter, in the OVF environment handed to the
// guest, and in an SSH key cloud-init installed from it, where both '/' had
// become &#x2F; so the key could never authenticate.
//
// Nothing reports any of this. The apply succeeds, the machine boots, the
// declared value never takes effect, and no later apply can repair it because
// the update path refuses to write that value back. Refusing at plan time
// cannot make such a value work — only the platform can — but it stops
// Terraform from reporting success for something that did not happen.
//
// A `seedfrom` URL is affected by construction, and so is the Base64 body of
// most SSH public keys.
func validateCloudInitMap() schema.SchemaValidateDiagFunc {
	keys := validation.MapKeyMatch(cloudInitKeys, `The following key is not allowed for cloud-init`)
	return func(i interface{}, p cty.Path) diag.Diagnostics {
		diags := keys(i, p)
		m, ok := i.(map[string]interface{})
		if !ok {
			return diags
		}
		names := make([]string, 0, len(m))
		for name := range m {
			names = append(names, name)
		}
		// Map iteration order is random; the diagnostics must not be.
		sort.Strings(names)
		for _, name := range names {
			s, ok := m[name].(string)
			if !ok || transparentVAppString(s) {
				continue
			}
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "cloud_init value cannot be stored faithfully",
				Detail:        fmt.Sprintf("The value of %q %s. The deployment API trims surrounding whitespace and HTML-escapes & < > \" ' / ` and \\, so the machine would receive a value different from the one declared — with no error reported, and no way to apply the difference afterwards.", name, cloudInitValueFault(s)),
				AttributePath: append(p, cty.IndexStep{Key: cty.StringVal(name)}),
			})
		}
		return diags
	}
}

// cloudInitValueFault names what makes a value untransmittable. An operator
// looking at three kilobytes of Base64 cannot find the offending character
// unaided, so the diagnostic points straight at it.
func cloudInitValueFault(s string) string {
	switch {
	case s == "":
		return "is empty"
	case strings.TrimSpace(s) != s:
		return "has leading or trailing whitespace"
	}
	if i := strings.IndexAny(s, "&<>\"'/`\\"); i >= 0 {
		return fmt.Sprintf("contains %q at offset %d", s[i], i)
	}
	return "cannot be transmitted without API normalization"
}

func transparentVAppString(s string) bool {
	return s != "" && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "&<>\"'/`\\")
}

// cloudInitPlatformEntities maps each entity form the platform was MEASURED to
// produce back to the character it replaced.
//
// Measured on a live machine by sending ONE string carrying all eight characters
// through each write path and reading back what was stored:
//
//	A/B&C<D>E"F\'G`H\\I
//
//	deployment (creation)    -> A/B&C&lt;D>E"F\'G`H\\I          only '<'
//	vApp properties (update) -> A&#x2F;B&amp;C&lt;D&gt;E&quot;F&#x27;G&#96;H&#x5C;I   all eight
//
// The two write paths do not agree, and an earlier measurement on another
// environment found the deployment path escaping '/' as well. That is precisely
// why the recognition below REVERSES the escaping instead of reproducing it:
// reproducing it would require knowing which subset a given path applied to a
// given value, and that is not a property of this provider. Reversing is
// indifferent to the subset.
var cloudInitPlatformEntities = []struct{ entity, char string }{
	{"&#x2F;", "/"},
	{"&lt;", "<"},
	{"&gt;", ">"},
	{"&quot;", "\""},
	{"&#x27;", "'"},
	{"&#96;", "`"},
	{"&#x5C;", "\\"},
	// &amp; LAST, and the order is load-bearing. A declared value containing a
	// literal "&lt;" is stored as "&amp;lt;"; un-escaping the ampersand first
	// would turn that into "&lt;" and then into "<", inventing a character the
	// operator never wrote and silently accepting a value as "our own".
	{"&amp;", "&"},
}

// cloudInitPlatformUnescape reverses the escaping above. It is the recognition
// half of the drift rule, not a refusal: a form that is NOT in the measured
// table is left alone, so the value still reports as drifted and fails closed.
func cloudInitPlatformUnescape(s string) string {
	for _, e := range cloudInitPlatformEntities {
		s = strings.ReplaceAll(s, e.entity, e.char)
	}
	return s
}

// cloudInitIsPlatformEscapeOf reports whether a live vApp value is what the
// platform produces from a recorded one: the recorded value trimmed, with any
// subset of the measured characters escaped. Both sides come from untyped maps —
// JSON for the live value, state for the recorded one — so a non-string on
// either side is simply not a match.
func cloudInitIsPlatformEscapeOf(live, recorded interface{}) bool {
	l, lok := live.(string)
	r, rok := recorded.(string)
	return lok && rok && cloudInitPlatformUnescape(l) == strings.TrimSpace(r)
}

// cloudInitDeployOptionsOverlap returns the keys declared in BOTH maps, sorted:
// map iteration order is random and must never reach a diagnostic.
func cloudInitDeployOptionsOverlap(cloudInit, deploy map[string]interface{}) []string {
	if len(cloudInit) == 0 || len(deploy) == 0 {
		return nil
	}
	var shared []string
	for key := range cloudInit {
		if _, both := deploy[key]; both {
			shared = append(shared, key)
		}
	}
	sort.Strings(shared)
	return shared
}

// cloudInitCollisionError is the single wording of the collision refusal, shared
// by the plan-time guard and the creation backstop so the two cannot drift apart.
func cloudInitCollisionError(shared []string) error {
	return fmt.Errorf(
		"%s declared in both cloud_init and deploy_options: creation sends the deploy_options value and the cloud_init one is never applied, so declare each key in exactly one of the two maps",
		strings.Join(shared, ", "))
}

// cloudInitCreateBackstop re-applies, on the RESOLVED configuration and before
// any platform call, the cloud_init rules the plan enforces — because every one
// of them fails open on a value the plan does not know, and creation has no
// apply-time copy of them.
//
// How the hole opens. terraform-plugin-sdk validates a configuration once, with
// the values the plan holds, and it does not validate an unknown one: a
// cloud_init whose value comes from another resource reaches validateCloudInitMap
// as unknown and is not checked at all. The collision CustomizeDiff has the same
// shape through d.Get, where an unresolved map reads as EMPTY and the overlap
// check is skipped. Neither runs again at apply.
//
// On UPDATE that costs nothing: planCloudInitEdits re-applies both rules at apply
// time, on resolved values, and refuses before any write. Creation had no such
// copy. computeVirtualMachineCreate merges cloud_init into the deploy options,
// gives deploy_options precedence on a shared key, and sends whatever it was
// given — so
//
//	cloud_init = { public-keys = tls_private_key.k.public_key_openssh }
//
// validated clean, deployed, and stored &#x2F; wherever the key held a '/',
// while state recorded the declared form and the refresh deliberately leaves
// such a key alone. Nothing ever reported it. That is exactly the silent false
// success #549 exists to remove, reaching the machine through the one door a
// plan cannot close.
//
// The key and value rules are not restated here: validateCloudInitMap IS the
// rule, called again on the resolved map, so the two copies cannot diverge.
func cloudInitCreateBackstop(d *schema.ResourceData) diag.Diagnostics {
	cloudInit, _ := d.Get("cloud_init").(map[string]interface{})
	if len(cloudInit) == 0 {
		return nil
	}
	path := cty.Path{cty.GetAttrStep{Name: "cloud_init"}}
	diags := validateCloudInitMap()(cloudInit, path)
	deploy, _ := d.Get("deploy_options").(map[string]interface{})
	if shared := cloudInitDeployOptionsOverlap(cloudInit, deploy); len(shared) > 0 {
		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "cloud_init key declared in deploy_options as well",
			Detail:        cloudInitCollisionError(shared).Error(),
			AttributePath: path,
		})
	}
	return diags
}

// Normalize only documented empty metadata and 0/1 booleans. Unknown metadata
// prevents replay; unrelated properties bypass this function and remain exact.
func vAppPayload(p map[string]interface{}) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	for k, v := range p {
		switch k {
		case "key":
			continue
		case "typeReference":
			if v == nil || v == float64(0) || v == "" {
				continue
			}
			return nil, fmt.Errorf("vApp typeReference cannot be preserved by the public API")
		case "userConfigurable":
			switch v {
			case true, float64(1):
				out[k] = true
			case false, float64(0):
				out[k] = false
			default:
				return nil, fmt.Errorf("invalid vApp userConfigurable metadata")
			}
		case "id", "label", "type", "classId", "instanceId", "category", "description", "defaultValue", "value":
			required := k == "id" || k == "label" || k == "type"
			if !required && (v == nil || v == float64(0) || v == "") {
				continue
			}
			s, ok := v.(string)
			if !ok || !transparentVAppString(s) {
				return nil, fmt.Errorf("vApp metadata cannot be replayed without API normalization")
			}
			if k != "value" && k != "defaultValue" && utf8.RuneCountInString(s) > 60 {
				return nil, fmt.Errorf("vApp metadata exceeds the public API limit")
			}
			out[k] = s
		default:
			return nil, fmt.Errorf("unknown vApp metadata cannot be preserved by the public API")
		}
	}
	if out["id"] == nil || out["label"] == nil || (out["type"] != "string" && out["type"] != "password") {
		return nil, fmt.Errorf("cloud_init updates require a labelled string or password vApp property")
	}
	return out, nil
}

func vAppPropertiesByKey(vm *client.VAppSnapshot) map[interface{}]map[string]interface{} {
	out := map[interface{}]map[string]interface{}{}
	for _, p := range vm.Properties {
		out[p["key"]] = p
	}
	return out
}

func sameVAppSnapshot(a, b *client.VAppSnapshot) bool {
	return a.ID == b.ID && b.PowerState == "stopped" && b.Enabled && reflect.DeepEqual(vAppPropertiesByKey(a), vAppPropertiesByKey(b))
}

func verifyVAppSnapshot(ctx context.Context, c *client.Client, id string, expected *client.VAppSnapshot) error {
	actual, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if !sameVAppSnapshot(expected, actual) {
		return fmt.Errorf("vApp snapshot changed")
	}
	return nil
}

func verifyVAppAddition(before, after *client.VAppSnapshot, payload map[string]interface{}) error {
	if before.ID != after.ID || after.PowerState != "stopped" || !after.Enabled || len(after.Properties) != len(before.Properties)+1 {
		return fmt.Errorf("unexpected addition result")
	}
	old := vAppPropertiesByKey(before)
	found := false
	for _, p := range after.Properties {
		if original, ok := old[p["key"]]; ok {
			if !reflect.DeepEqual(original, p) {
				return fmt.Errorf("existing descriptor changed")
			}
			continue
		}
		actual, err := vAppPayload(p)
		if err != nil || found || !reflect.DeepEqual(actual, payload) {
			return fmt.Errorf("added descriptor differs")
		}
		found = true
	}
	if !found {
		return fmt.Errorf("added descriptor missing")
	}
	return nil
}

// refreshVMwareCloudInit resynchronises the tracked cloud_init keys from the
// live vApp. Its contract is deliberately narrow, because this runs on every
// plan and must never turn an unrelated condition into a failed one:
//
//   - it never ADDS a key, so a property the operator never declared — from the
//     image, from deploy_options, from the platform — is never adopted;
//   - it never REMOVES a key. Absence is not evidence of deletion: the clone and
//     from-scratch creation paths send no deploy options at all, so an ordinary
//     machine can track keys its vApp never received. Dropping those would make
//     an unchanged configuration plan, and that plan is then refused outright on
//     a running machine without allow_vm_restart;
//   - it never touches a key that deploy_options also declares. Creation sends
//     the deploy_options value for a shared key, so the live property belongs to
//     that attribute; recording it here would leave a diff the planner refuses
//     for collision, for ever;
//   - it never touches a key whose RECORDED value this provider could not write
//     back. Creation refuses such a value since 1.13.0, but state written by an
//     earlier release can still hold one, and the platform stored a deformed
//     version of it, so such a key would gain a diff that can never be applied;
//   - it never FAILS. Every anomaly degrades to a warning and leaves the
//     recorded value in place.
//
// What remains is the single case it exists for: a key Terraform owns, whose
// live value is observably different and whose recorded value it could restore.
func refreshVMwareCloudInit(ctx context.Context, d *schema.ResourceData, c *client.Client) diag.Diagnostics {
	tracked, ok := d.Get("cloud_init").(map[string]interface{})
	if !ok || len(tracked) == 0 {
		// Nothing is tracked, so there is nothing to resynchronise — and nothing
		// to adopt either. An imported machine stays in this branch.
		return nil
	}
	deploy, _ := d.Get("deploy_options").(map[string]interface{})
	live, err := c.Compute().VirtualMachine().ReadVAppSnapshot(ctx, d.Id())
	if err != nil {
		return cloudInitRefreshWarning("the vApp properties could not be read, so the recorded cloud_init values were left untouched")
	}
	var diags diag.Diagnostics
	observed := make(map[string]interface{}, len(tracked))
	for key, was := range tracked {
		// The recorded value is the default outcome for every key. Only a value
		// positively observed on a property this attribute owns replaces it.
		observed[key] = was
		if _, shared := deploy[key]; shared {
			continue
		}
		// A value this provider could never write back is not worth a diff.
		// Creation refuses such a value since 1.13.0 — at plan time, and again
		// on the resolved configuration for a value the plan could not see — but
		// state written by an EARLIER release can still hold one, and the
		// platform deformed it on the way in: it trims surrounding whitespace
		// and HTML-escapes '/' as &#x2F;. Resynchronising such a key would
		// record the deformed form, raise a diff against the declared one, and
		// then refuse to apply it — permanently, and on a running machine the
		// plan itself would fail. Leaving it alone is what 1.12 did, and it
		// converged.
		//
		// The deformation is ON THE MACHINE, not only in the copy the API
		// reports back: it was measured in vCenter, in the OVF environment
		// handed to the guest, and in an SSH key cloud-init installed from such
		// a value. That is why creation refuses these values outright, and why
		// the way out for a machine that already carries one is the planner
		// recognising the escaped form (see cloudInitPlatformUnescape) rather
		// than anything this refresh could record.
		if s, ok := was.(string); !ok || !transparentVAppString(s) {
			continue
		}
		var found map[string]interface{}
		duplicated := false
		for _, p := range live.Properties {
			if p["id"] != key {
				continue
			}
			if found != nil {
				duplicated = true
				break
			}
			found = p
		}
		if duplicated {
			// Which of the two is the managed one cannot be decided here, and
			// guessing would record a value that was never applied.
			diags = append(diags, cloudInitRefreshWarning(fmt.Sprintf("the vApp holds duplicate entries for the cloud_init key %q, so its recorded value was left untouched", key))...)
			continue
		}
		if found == nil {
			continue
		}
		value, ok := found["value"].(string)
		if !ok {
			diags = append(diags, cloudInitRefreshWarning(fmt.Sprintf("the vApp property for the cloud_init key %q does not hold a string value, so its recorded value was left untouched", key))...)
			continue
		}
		observed[key] = value
	}
	if err := d.Set("cloud_init", observed); err != nil {
		return append(diags, cloudInitRefreshWarning("the observed cloud_init values could not be recorded")...)
	}
	return diags
}

// cloudInitChangeIsOwnedByDeployOptions reports whether EVERY cloud_init key
// this plan changes is also declared in deploy_options.
//
// The update planner skips such a key — it neither writes nor deletes it,
// because deploy_options owns the live value — so the apply performs no vApp
// write and needs no quiescent machine. Without this, the only non-destructive
// way out of a collision (dropping the key from cloud_init) would demand a
// restart opt-in on a running machine, to authorise an edit that touches
// nothing.
//
// It returns false unless something actually changes, so it can never turn a
// no-op plan into an exemption, and false as soon as one changed key is NOT
// deploy-owned: a mixed change still writes to the vApp.
func cloudInitChangeIsOwnedByDeployOptions(d *schema.ResourceDiff) bool {
	deploy, _ := d.Get("deploy_options").(map[string]interface{})
	if len(deploy) == 0 {
		return false
	}
	oldRaw, newRaw := d.GetChange("cloud_init")
	oldMap, _ := oldRaw.(map[string]interface{})
	newMap, _ := newRaw.(map[string]interface{})
	changed := false
	for key, was := range oldMap {
		if want, present := newMap[key]; !present || want != was {
			if _, owned := deploy[key]; !owned {
				return false
			}
			changed = true
		}
	}
	for key, want := range newMap {
		if was, present := oldMap[key]; !present || was != want {
			if _, owned := deploy[key]; !owned {
				return false
			}
			changed = true
		}
	}
	return changed
}

// cloudInitChangeSet is the slice of ResourceData and ResourceDiff the sole-
// change rule needs, so the same rule can be evaluated at plan time and at apply
// time instead of being written twice and drifting apart.
type cloudInitChangeSet interface {
	HasChange(string) bool
}

// cloudInitOtherChangedAttribute names an attribute that would change in the
// same apply as cloud_init, or "" when none would.
//
// Only the two power-intent attributes may travel with cloud_init: this path
// drives power itself, and allow_vm_restart is the opt-in for doing so, so
// refusing it would make the opt-in impossible to grant in the same apply as
// the change it authorises. Everything else is refused, because the vApp
// add/delete sequence is non-atomic and assumes a quiescent machine.
//
// The names are sorted before one is picked. Schema iteration order is random,
// so without this the diagnostic named a different attribute from run to run
// when several had changed — which makes a refusal impossible to script against
// and a test impossible to pin.
func cloudInitOtherChangedAttribute(d cloudInitChangeSet) string {
	var changed []string
	for name, s := range resourceVirtualMachine().Schema {
		switch name {
		case "cloud_init", "power_state", "allow_vm_restart":
			continue
		}
		if (s.Optional || s.Required) && d.HasChange(name) {
			changed = append(changed, name)
		}
	}
	if len(changed) == 0 {
		return ""
	}
	sort.Strings(changed)
	return changed[0]
}

// cloudInitKnownChanges reports a change only once Terraform has resolved the
// new value. At plan time an unresolved attribute reports as changed whether or
// not it will be, so taking it at face value would refuse a legitimate plan;
// the apply-time check sees the resolved value and decides for real.
type cloudInitKnownChanges struct{ d *schema.ResourceDiff }

func (k cloudInitKnownChanges) HasChange(name string) bool {
	return k.d.NewValueKnown(name) && k.d.HasChange(name)
}

// cloudInitRefuse freezes the state so a failed update cannot persist planned
// values — the SDK's default on error — EXCEPT where a live read has positively
// contradicted the planned power state. Freezing there would keep the one
// answer the provider has just ruled out: "running", for a machine it powered
// off itself and is deliberately leaving down.
//
// Everything else this path may change is handled on its own: cloud_init by the
// rollback deferred in updateVMwareCloudInit, power_state by the deferred that
// records run.observed, and no other attribute can travel with cloud_init in an
// in-place update.
func cloudInitRefuse(d *schema.ResourceData, run *cloudInitRun, diags diag.Diagnostics) diag.Diagnostics {
	if run.observed != "" && run.observed != d.Get("power_state").(string) {
		return diags
	}
	return refuseBeforeAnyMutation(d, diags)
}

// cloudInitRecordPower stores a live power observation in the schema's
// vocabulary. An unrecognised or transitional state records nothing: it is not
// evidence of either stable state, and guessing is what this path exists to
// avoid.
func cloudInitRecordPower(run *cloudInitRun, live string) {
	switch live {
	case "running":
		run.observed = "on"
	case "stopped":
		run.observed = "off"
	}
}

// cloudInitRefreshWarning degrades a refresh anomaly to a warning. A refresh
// that fails takes the whole plan with it and leaves the operator no provider
// side recovery, which is too blunt for a step that only ever corrects values.
func cloudInitRefreshWarning(detail string) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "cloud_init was not resynchronised from the vApp",
		Detail:   detail,
	}}
}

// computeVirtualMachineReadRefresh is the resource's ReadContext. It is the
// general read plus the cloud_init resynchronisation above, kept separate so
// that create and update — which both end in computeVirtualMachineRead — keep
// exactly the behaviour they had.
func computeVirtualMachineReadRefresh(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	diags := computeVirtualMachineRead(ctx, d, meta)
	if diags.HasError() || d.Id() == "" {
		// A removed or unreadable resource has no vApp worth re-reading.
		return diags
	}
	return append(diags, refreshVMwareCloudInit(ctx, d, getClient(meta))...)
}
