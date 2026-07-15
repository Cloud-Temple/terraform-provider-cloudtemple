package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// NOTE (C5): this resource manages ONLY the FIP<->static-IP binding. The floating
// IP is provisioned out-of-band (cloudtemple_vpc_floating_ip) and is left intact on
// destroy — Unbind removes the relationship, never the IP. Both ids are ForceNew, so
// there is no Update (a change of either id = unbind then re-bind = recreate).
//
// State-safety doctrine (carried from C3/C4, enforced by the by-id oracle
// client.ResolveBinding, NOT by retry):
//   - never CLOBBER: create binds ONLY from a provably-free (Unbound) state; a FIP
//     bound elsewhere (BoundToOther) or an inconclusive read FAILS CLOSED.
//   - never falsely report success: create AND delete confirm by a positive read.
//   - never DROP on inconclusive evidence: refresh drops only on positive proof the
//     binding is gone (Unbound / BoundToOther / authoritative 404).
//   - fail closed on a transient activity failure (no naive retry); a single
//     confirm-read salvages the case where the activity actually landed (#319 carries
//     a full proof-gated retry as a separate follow-up).

func resourceVPCFloatingIPBinding() *schema.Resource {
	return &schema.Resource{
		Description: "Bind a VPC floating (public) IP to a static IP — the day-to-day \"expose a VM publicly\" action (VM adapter MAC -> static IP -> floating IP). Reversible by construction: the floating IP is provisioned out-of-band (cloudtemple_vpc_floating_ip) and is left intact on destroy; only the binding is created/removed. Changing either id forces a new binding.",

		CreateContext: resourceVPCFloatingIPBindingCreate,
		ReadContext:   resourceVPCFloatingIPBindingRead,
		DeleteContext: resourceVPCFloatingIPBindingDelete,
		// No UpdateContext: both inputs are ForceNew.
		Importer: &schema.ResourceImporter{StateContext: resourceVPCFloatingIPBindingImport},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"floating_ip_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the floating (public) IP to bind. Changing this forces a new binding.",
			},
			"static_ip_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the static IP to bind the floating IP to. Changing this forces a new binding.",
			},
			"floating_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The public IP address of the bound floating IP.",
			},
		},
	}
}

// floatingIPBindingReadMode mirrors the static/floating IP read modes: a CONFIRMED
// "binding gone" (Unbound / BoundToOther / authoritative 404) drops the resource on
// REFRESH but FAILS CLOSED right after a write (the binding was just established;
// an unconfirmed read is eventual consistency, never a deletion).
type floatingIPBindingReadMode int

const (
	bindingReadForRefresh floatingIPBindingReadMode = iota
	bindingReadAfterWrite
)

// vpcFloatingIPBindingFuncs abstracts the binding API surface so the CRUD
// orchestration is unit tested without HTTP calls.
type vpcFloatingIPBindingFuncs struct {
	resolve     func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, *client.FloatingIP, bool, error)
	bind        func(ctx context.Context, fipID, staticID string) (activityID string, err error)
	unbind      func(ctx context.Context, fipID, staticID string) (activityID string, err error)
	wait        vpcActivityWaitFunc
	isTransient func(error) bool
}

func resourceVPCFloatingIPBindingClientFuncs(meta any) vpcFloatingIPBindingFuncs {
	c := getClient(meta)
	return vpcFloatingIPBindingFuncs{
		resolve: c.VPC().FloatingIP().ResolveBinding,
		bind:    c.VPC().FloatingIP().Bind,
		unbind:  c.VPC().FloatingIP().Unbind,
		wait: func(ctx context.Context, activityID string) error {
			_, err := c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx))
			return err
		},
		isTransient: client.IsTransientActivityFailure,
	}
}

func resourceVPCFloatingIPBindingCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return createVPCFloatingIPBindingWith(ctx, d, resourceVPCFloatingIPBindingClientFuncs(meta))
}

func resourceVPCFloatingIPBindingRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return readVPCFloatingIPBindingInto(ctx, d, c.VPC().FloatingIP().ResolveBinding, bindingReadForRefresh)
}

func resourceVPCFloatingIPBindingDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	return deleteVPCFloatingIPBindingWith(ctx, d, resourceVPCFloatingIPBindingClientFuncs(meta))
}

// bindingResolveFunc is the by-id oracle signature shared by the read/create/delete
// logic (so they cannot drift on how the 4-state result is interpreted).
type bindingResolveFunc func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, *client.FloatingIP, bool, error)

// bindingID is the composite resource id "{floating_ip_id}/{static_ip_id}".
func bindingID(fipID, staticID string) string { return fipID + "/" + staticID }

// bindingUnbindConfirmed is the EXPLICIT delete success predicate. A delete is
// confirmed ONLY when the floating IP is authoritatively absent (404), or is present
// and Unbound, or is present and bound to a DIFFERENT static IP. BoundToTarget and
// Inconclusive are NOT confirmed (Inconclusive is "not BoundToTarget" but it never
// confirms a destructive decision — fail closed).
func bindingUnbindConfirmed(state client.FloatingIPBindingState, found bool) bool {
	if !found {
		return true // authoritative 404: the FIP is gone, so the binding is gone.
	}
	return state == client.FloatingIPBindingUnbound || state == client.FloatingIPBindingBoundToOther
}

// bindingStateLabel renders a binding state for diagnostics.
func bindingStateLabel(state client.FloatingIPBindingState, found bool) string {
	if !found {
		return "floating IP absent (404)"
	}
	switch state {
	case client.FloatingIPBindingUnbound:
		return "unbound"
	case client.FloatingIPBindingBoundToTarget:
		return "bound to the target static IP"
	case client.FloatingIPBindingBoundToOther:
		return "bound to a different static IP"
	default:
		return "inconclusive"
	}
}

// createVPCFloatingIPBindingWith holds the testable create logic. The worst outcome
// is a CLOBBER (binding a FIP that is bound elsewhere) or a falsely-reported success,
// so the pre-read gates binding and the post-read confirms it.
func createVPCFloatingIPBindingWith(ctx context.Context, d *schema.ResourceData, funcs vpcFloatingIPBindingFuncs) diag.Diagnostics {
	fipID := d.Get("floating_ip_id").(string)
	staticID := d.Get("static_ip_id").(string)

	// 1. Anti-clobber pre-read.
	state, _, found, err := funcs.resolve(ctx, fipID, staticID)
	if err != nil {
		return diag.Errorf("failed to bind floating IP %s to static IP %s: the pre-bind read failed (%s); refusing to bind on inconclusive evidence", fipID, staticID, err)
	}
	switch {
	case state == client.FloatingIPBindingBoundToTarget:
		// Already our pair (idempotent): no bind, just adopt and confirm.
		d.SetId(bindingID(fipID, staticID))
		return readVPCFloatingIPBindingInto(ctx, d, funcs.resolve, bindingReadAfterWrite)
	case found && state == client.FloatingIPBindingUnbound:
		// Provably free -> proceed to bind.
	default:
		// BoundToOther, Inconclusive, or an authoritative 404 (FIP gone) -> never bind.
		return diag.Errorf("refusing to bind floating IP %s to static IP %s: the floating IP is not provably free (%s). Unbind it elsewhere first, resolve an inconclusive read, or check the floating IP exists.", fipID, staticID, bindingStateLabel(state, found))
	}

	// 2. Bind (async). A non-transient bind error fails WITHOUT setting an id.
	activityID, berr := funcs.bind(ctx, fipID, staticID)
	if berr != nil {
		return diag.Errorf("failed to bind floating IP %s to static IP %s: %s", fipID, staticID, berr)
	}

	// 3. Wait — UNLESS activityID is empty (409 idempotent no-op: nothing to wait on).
	//    On a transient activity failure, ONE confirm-read salvages a landed bind; any
	//    other failure (or a not-landed transient) fails closed with NO id set.
	if activityID != "" {
		if werr := funcs.wait(ctx, activityID); werr != nil {
			if !funcs.isTransient(werr) {
				return diag.Errorf("failed to bind floating IP %s to static IP %s: %s", fipID, staticID, werr)
			}
			if st, _, fnd, cerr := funcs.resolve(ctx, fipID, staticID); cerr != nil || st != client.FloatingIPBindingBoundToTarget || !fnd {
				return diag.Errorf("failed to bind floating IP %s to static IP %s: the bind activity hit a transient failure and the binding is not confirmed; re-run `terraform apply`. (%s)", fipID, staticID, werr)
			}
			// Landed despite the transient -> fall through.
		}
	}

	// 4. The bind landed (sync 409, async completion, or transient-but-confirmed).
	d.SetId(bindingID(fipID, staticID))

	// 5. Confirm by read (readAfterWrite: never drop the just-set id; success only on
	//    a positive BoundToTarget).
	return readVPCFloatingIPBindingInto(ctx, d, funcs.resolve, bindingReadAfterWrite)
}

// readVPCFloatingIPBindingInto holds the testable read logic. The resource is NEVER
// dropped on an inconclusive or errored read; in refresh it drops ONLY on positive
// proof the binding is gone (Unbound / BoundToOther / authoritative 404).
func readVPCFloatingIPBindingInto(ctx context.Context, d *schema.ResourceData, resolve bindingResolveFunc, mode floatingIPBindingReadMode) diag.Diagnostics {
	fipID := d.Get("floating_ip_id").(string)
	staticID := d.Get("static_ip_id").(string)

	state, fip, found, err := resolve(ctx, fipID, staticID)
	if err != nil {
		return diag.Errorf("failed to read the floating IP %s <-> static IP %s binding: %s. The resource is kept in the state (a forbidden, partial, or inconsistent read is not proof of absence); resolve the error, then refresh.", fipID, staticID, err)
	}

	if mode == bindingReadAfterWrite {
		// Just written: positive existence evidence. Success ONLY on BoundToTarget;
		// anything else keeps the id and errors (never drop a just-written id, never
		// report success on an unconfirmed bind).
		if found && state == client.FloatingIPBindingBoundToTarget {
			return flattenFloatingIPBinding(d, fip)
		}
		return diag.Errorf("the floating IP %s <-> static IP %s binding was just written but is not yet confirmed bound (%s); the resource is kept in the state with its id. Re-run `terraform apply` or `terraform refresh`.", fipID, staticID, bindingStateLabel(state, found))
	}

	// Refresh.
	switch {
	case !found:
		// Authoritative 404: the floating IP is gone, so the binding is gone.
		d.SetId("")
		return nil
	case state == client.FloatingIPBindingBoundToTarget:
		return flattenFloatingIPBinding(d, fip)
	case state == client.FloatingIPBindingUnbound, state == client.FloatingIPBindingBoundToOther:
		// Positive evidence the binding is gone (FIP now free, or rebound elsewhere).
		d.SetId("")
		return nil
	default:
		// Inconclusive -> fail closed (keep; never drop on inconclusive evidence).
		return diag.Errorf("the floating IP %s <-> static IP %s binding could not be confirmed (inconclusive read); the resource is kept in the state to avoid a wrong removal. Resolve the read, then refresh.", fipID, staticID)
	}
}

func flattenFloatingIPBinding(d *schema.ResourceData, fip *client.FloatingIP) diag.Diagnostics {
	sw := newStateWriter(d)
	if fip != nil {
		sw.set("floating_ip_address", fip.IPAddress)
	}
	return sw.diags
}

// deleteVPCFloatingIPBindingWith holds the testable delete logic. It NEVER reports a
// successful delete without positive confirmation (the unbind-confirmed predicate),
// and never unbinds a pair it cannot prove is ours.
func deleteVPCFloatingIPBindingWith(ctx context.Context, d *schema.ResourceData, funcs vpcFloatingIPBindingFuncs) diag.Diagnostics {
	fipID := d.Get("floating_ip_id").(string)
	staticID := d.Get("static_ip_id").(string)

	// 1. Preflight: only unbind a provably-ours pair; already-gone is idempotent;
	//    inconclusive/error fails closed.
	state, _, found, err := funcs.resolve(ctx, fipID, staticID)
	if err != nil {
		return diag.Errorf("failed to unbind floating IP %s from static IP %s: the preflight read failed (%s); the resource is kept (refusing to unbind on inconclusive evidence)", fipID, staticID, err)
	}
	if bindingUnbindConfirmed(state, found) {
		return nil // already not our pair: idempotent success, NO unbind issued.
	}
	if state != client.FloatingIPBindingBoundToTarget {
		// Inconclusive (found, not a definite state) -> fail closed.
		return diag.Errorf("the floating IP %s <-> static IP %s binding could not be confirmed before unbinding (%s); the resource is kept to avoid a wrong removal. Resolve the read, then destroy.", fipID, staticID, bindingStateLabel(state, found))
	}

	// 2. Unbind (async). On error, the client surfaces 404/403 for US to confirm: a
	//    read proving the pair is gone is an idempotent success; otherwise fail closed.
	activityID, uerr := funcs.unbind(ctx, fipID, staticID)
	if uerr != nil {
		if st, _, fnd, cerr := funcs.resolve(ctx, fipID, staticID); cerr == nil && bindingUnbindConfirmed(st, fnd) {
			return nil
		}
		return diag.Errorf("failed to unbind floating IP %s from static IP %s: %s (and the unbind could not be confirmed by read); the resource is kept", fipID, staticID, uerr)
	}

	// 3. Wait — UNLESS activityID is empty (409 idempotent). A transient failure is
	//    salvaged by a single confirm-read; otherwise fail closed.
	if activityID != "" {
		if werr := funcs.wait(ctx, activityID); werr != nil {
			if funcs.isTransient(werr) {
				if st, _, fnd, cerr := funcs.resolve(ctx, fipID, staticID); cerr == nil && bindingUnbindConfirmed(st, fnd) {
					return nil
				}
			}
			return diag.Errorf("failed to unbind floating IP %s from static IP %s: the unbind activity did not complete: %s; the resource is kept", fipID, staticID, werr)
		}
	}

	// 4. Final positive confirmation (the explicit predicate).
	st, _, fnd, cerr := funcs.resolve(ctx, fipID, staticID)
	if cerr != nil {
		return diag.Errorf("the unbind of floating IP %s from static IP %s could not be confirmed (read failed: %s); the resource is kept", fipID, staticID, cerr)
	}
	if bindingUnbindConfirmed(st, fnd) {
		return nil
	}
	return diag.Errorf("the unbind of floating IP %s from static IP %s did not take (%s); the resource is kept in the state.", fipID, staticID, bindingStateLabel(st, fnd))
}

// resourceVPCFloatingIPBindingImport accepts "{floating_ip_id}/{static_ip_id}" and
// seeds both ids so the subsequent Read can corroborate the pair.
func resourceVPCFloatingIPBindingImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid import id %q for cloudtemple_vpc_floating_ip_binding; expected \"<floating_ip_id>/<static_ip_id>\"", d.Id())
	}
	if err := d.Set("floating_ip_id", parts[0]); err != nil {
		return nil, err
	}
	if err := d.Set("static_ip_id", parts[1]); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
