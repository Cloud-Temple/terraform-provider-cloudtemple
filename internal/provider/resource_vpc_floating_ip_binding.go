package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// vpcFloatingIPBindingIDSeparator joins the floating IP id and the static IP id
// into the resource id "{floatingIpId}:{staticIpId}". A UUID never contains a
// ":", so the split is unambiguous.
const vpcFloatingIPBindingIDSeparator = ":"

// maxFloatingIPBindingConfirmAttempts bounds the number of post-mutation
// confirmation reads (bind convergence; stable-negative ruling on read). It is a
// named constant — never a magic number inline — and the inter-attempt wait is
// an injectable seam so unit tests run with zero real sleep.
const maxFloatingIPBindingConfirmAttempts = 6

func resourceVPCFloatingIPBinding() *schema.Resource {
	return &schema.Resource{
		Description: "Bind a pre-existing VPC floating IP to a VPC static IP. The floating IP must already be provisioned out-of-band: this resource does NOT create or destroy the floating IP, nor manage its description — it only manages the association (create = bind, delete = unbind). Both attributes force a new resource.",

		CreateContext: resourceVPCFloatingIPBindingCreate,
		ReadContext:   resourceVPCFloatingIPBindingRead,
		DeleteContext: resourceVPCFloatingIPBindingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVPCFloatingIPBindingImport,
		},

		Schema: map[string]*schema.Schema{
			// In
			"floating_ip_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the pre-existing floating IP to bind. Changing this forces a new resource.",
			},
			"static_ip_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the static IP to bind the floating IP to. Changing this forces a new resource.",
			},

			// Out
			"floating_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The address of the bound floating IP.",
			},
			"static_ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The address of the static IP the floating IP is bound to.",
			},
		},
	}
}

// vpcFloatingIPBindingFuncs is the injected API surface used by the testable
// inner functions, so the CRUD logic is unit tested without HTTP calls or real
// sleeps (mirrors the openiaas_vif_retry seam).
type vpcFloatingIPBindingFuncs struct {
	// read returns the live floating IP (nil when absent/403/ambiguous).
	read func(ctx context.Context, fipID string) (*client.FloatingIP, error)
	// bind starts the bind and returns the activity id ("" on a 409 idempotent).
	bind func(ctx context.Context, fipID, staticID string) (string, error)
	// unbind starts the unbind and returns the activity id ("" on a 409 idempotent).
	unbind func(ctx context.Context, fipID, staticID string) (string, error)
	// wait waits for an activity to complete (skipped when the id is empty).
	wait func(ctx context.Context, activityID string) error
	// corroborate strictly classifies the FIP/static relationship from a
	// complete 200 listing.
	corroborate func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error)
	// sleep waits between confirmation attempts; returns ctx.Err() on cancel.
	sleep func(ctx context.Context, attempt int) error
	// retrySleep backs off between transient-failure retries of the async
	// write (bind/unbind). Defaults to defaultVPCSleep (attempt*10s) when nil.
	// Distinct from sleep, which paces the post-mutation confirmation reads.
	retrySleep func(ctx context.Context, attempt int) error
	// isTransient classifies an activity failure as a retryable transient
	// platform error. Defaults to client.IsTransientActivityFailure when nil.
	isTransient func(err error) bool
}

// classifyFloatingIPBindingLive resolves the live binding state of fipID
// relative to staticID, using the per-id read first and falling back to the
// strict listing when that read is ambiguous (nil/403/mismatched-id). It
// mirrors the create pre-bind evidence rules so a transient-failure retry
// re-classifies on the same basis before re-issuing a write. A read error is
// returned as-is (it is NOT an activity failure, so the retry helper stops on
// it) rather than acting on stale evidence.
func classifyFloatingIPBindingLive(ctx context.Context, fipID, staticID string, funcs vpcFloatingIPBindingFuncs) (client.FloatingIPBindingState, error) {
	rawFip, err := funcs.read(ctx, fipID)
	if err != nil {
		return client.FloatingIPBindingInconclusive, err
	}
	if fip, ok := usableFloatingIPRead(rawFip, fipID); ok {
		return client.ClassifyFloatingIPBinding(fip, staticID), nil
	}
	return funcs.corroborate(ctx, fipID, staticID)
}

// usableFloatingIPRead applies the #312 R7 id-match guard to a per-id read
// (GET /floating_ips/{fipID}): it returns the floating IP ONLY when the response
// is a non-nil body carrying EXACTLY fipID. A nil read (absent/403/ambiguous), an
// empty id, or a DIFFERENT id are all collapsed to (nil, false): such a body must
// NEVER be used as positive evidence, because a structurally incomplete or
// mismatched 200 would otherwise let create bind, read drop, delete accept "gone",
// or import succeed on a body that does not actually describe fipID.
//
// Callers treat (nil, false) exactly like a nil/ambiguous read (fall to the
// strict-listing corroboration, keep state, or fail closed — per their own rule).
func usableFloatingIPRead(fip *client.FloatingIP, fipID string) (*client.FloatingIP, bool) {
	if fip == nil || fip.ID == "" || fip.ID != fipID {
		return nil, false
	}
	return fip, true
}

func defaultFloatingIPBindingSleep(ctx context.Context, attempt int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(attempt) * time.Second):
		return nil
	}
}

// clientFloatingIPBindingFuncs wires the inner functions to the real API client.
func clientFloatingIPBindingFuncs(c *client.Client, options *client.WaiterOptions) vpcFloatingIPBindingFuncs {
	return vpcFloatingIPBindingFuncs{
		read: func(ctx context.Context, fipID string) (*client.FloatingIP, error) {
			return c.VPC().FloatingIP().Read(ctx, fipID)
		},
		bind: func(ctx context.Context, fipID, staticID string) (string, error) {
			return c.VPC().FloatingIP().Bind(ctx, fipID, staticID)
		},
		unbind: func(ctx context.Context, fipID, staticID string) (string, error) {
			return c.VPC().FloatingIP().Unbind(ctx, fipID, staticID)
		},
		wait: func(ctx context.Context, activityID string) error {
			if activityID == "" {
				// A 409-idempotent bind/unbind produced no activity to wait on.
				return nil
			}
			_, err := c.Activity().WaitForCompletion(ctx, activityID, options)
			return err
		},
		corroborate: func(ctx context.Context, fipID, staticID string) (client.FloatingIPBindingState, error) {
			return c.VPC().FloatingIP().CorroborateBinding(ctx, fipID, staticID)
		},
	}
}

func resourceVPCFloatingIPBindingCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	return createVPCFloatingIPBinding(ctx, d,
		d.Get("floating_ip_id").(string), d.Get("static_ip_id").(string),
		clientFloatingIPBindingFuncs(c, getWaiterOptions(ctx)))
}

func resourceVPCFloatingIPBindingRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	fipID, staticID, err := splitFloatingIPBindingID(d.Id())
	if err != nil {
		return diag.Errorf("invalid VPC floating IP binding id %q: %s", d.Id(), err)
	}
	return readVPCFloatingIPBinding(ctx, d, fipID, staticID, clientFloatingIPBindingFuncs(c, getWaiterOptions(ctx)))
}

func resourceVPCFloatingIPBindingDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	fipID, staticID, err := splitFloatingIPBindingID(d.Id())
	if err != nil {
		return diag.Errorf("invalid VPC floating IP binding id %q: %s", d.Id(), err)
	}
	return deleteVPCFloatingIPBinding(ctx, d, fipID, staticID, clientFloatingIPBindingFuncs(c, getWaiterOptions(ctx)))
}

func resourceVPCFloatingIPBindingImport(ctx context.Context, d *schema.ResourceData, meta any) ([]*schema.ResourceData, error) {
	c := getClient(meta)
	return importVPCFloatingIPBinding(ctx, d, clientFloatingIPBindingFuncs(c, getWaiterOptions(ctx)))
}

// importVPCFloatingIPBinding holds the testable import logic. It parses the
// composite id strictly into two non-empty halves, then CONFIRMS the pair
// actually exists (a phantom import — a FIP not bound to this static IP — must
// fail rather than enter the state).
func importVPCFloatingIPBinding(ctx context.Context, d *schema.ResourceData, funcs vpcFloatingIPBindingFuncs) ([]*schema.ResourceData, error) {
	fipID, staticID, err := splitFloatingIPBindingID(d.Id())
	if err != nil {
		return nil, fmt.Errorf("invalid VPC floating IP binding id %q: %w (expected \"{floating_ip_id}%s{static_ip_id}\")", d.Id(), err, vpcFloatingIPBindingIDSeparator)
	}

	rawFip, err := funcs.read(ctx, fipID)
	if err != nil {
		return nil, fmt.Errorf("failed to read floating IP %s while importing the binding: %w", fipID, err)
	}
	// The per-id read must carry EXACTLY fipID; a nil/403 or a mismatched/empty-id
	// body cannot confirm the pair and must fail the import (no phantom) (#312 R7).
	fip, ok := usableFloatingIPRead(rawFip, fipID)
	if !ok {
		return nil, fmt.Errorf("cannot import VPC floating IP binding %q: floating IP %s could not be read consistently (absent, access denied, or an id-inconsistent body)", d.Id(), fipID)
	}
	if fip.StaticIP == nil || fip.StaticIP.ID != staticID {
		return nil, fmt.Errorf("cannot import VPC floating IP binding %q: floating IP %s is not bound to static IP %s", d.Id(), fipID, staticID)
	}

	if err := d.Set("floating_ip_id", fipID); err != nil {
		return nil, err
	}
	if err := d.Set("static_ip_id", staticID); err != nil {
		return nil, err
	}
	writeFloatingIPBindingComputed(d, fip)
	return []*schema.ResourceData{d}, nil
}

// createVPCFloatingIPBinding holds the testable create logic. It is FAIL-CLOSED:
// the binding is established (or adopted) only on positive HTTP 200 evidence,
// and the resource id is set ONLY after the binding is confirmed converged.
func createVPCFloatingIPBinding(ctx context.Context, d *schema.ResourceData, fipID, staticID string, funcs vpcFloatingIPBindingFuncs) diag.Diagnostics {
	if funcs.sleep == nil {
		funcs.sleep = defaultFloatingIPBindingSleep
	}

	// 1. Pre-bind read. Mutate only on positive 200 evidence. The per-id read is
	//    only usable as evidence when it carries EXACTLY fipID (#312 R7 guard); a
	//    nil/403/mismatched-id read is routed like an ambiguous read.
	rawFip, err := funcs.read(ctx, fipID)
	if err != nil {
		return diag.Errorf("failed to read floating IP %s before binding: %s", fipID, err)
	}
	if fip, ok := usableFloatingIPRead(rawFip, fipID); ok {
		switch client.ClassifyFloatingIPBinding(fip, staticID) {
		case client.FloatingIPBindingBoundToTarget:
			// Already bound to OUR static IP: ADOPT, do not re-bind.
			d.SetId(makeFloatingIPBindingID(fipID, staticID))
			writeFloatingIPBindingComputed(d, fip)
			return nil
		case client.FloatingIPBindingBoundToOther:
			// Bound to a DIFFERENT static IP (non-empty different id): anti-clobber,
			// ZERO bind POST.
			return diag.Errorf(
				"floating IP %s is already bound to static IP %s; refusing to clobber it by binding it to %s. Unbind it first (or import the existing binding).",
				fipID, fip.StaticIP.ID, staticID,
			)
		case client.FloatingIPBindingInconclusive:
			// Present, but the nested staticIp is structurally incomplete (a staticIp
			// object with an empty/omitted id). This is NOT proof the FIP is free, nor
			// proof it is bound elsewhere: it must NOT unlock a bind and must NOT be
			// reported as "bound to a different static IP". Fail closed with ZERO bind
			// POST so we never clobber an out-of-band binding on an inconclusive read.
			return diag.Errorf(
				"floating IP %s was read with a structurally incomplete nested staticIp (present but id-less); its binding state is inconclusive, so refusing to bind it to %s to avoid clobbering a possible out-of-band binding. Resolve the inconsistent read, then retry.",
				fipID, staticID,
			)
		}
		// fip present and unbound -> fall through to bind (FIP provably free).
	} else {
		// nil/403/ambiguous OR a mismatched/empty-id body: a single read cannot
		// prove the FIP is free. A strict-200 listing MAY unlock the bind, but
		// ONLY when it positively shows the FIP present-and-UNBOUND, or already
		// present-and-our-pair. "Bound to other" is FAIL-CLOSED anti-clobber with
		// ZERO bind POST; "absent"/inconclusive never unlocks the bind.
		state, cerr := funcs.corroborate(ctx, fipID, staticID)
		if cerr != nil {
			return diag.Errorf(
				"floating IP %s could not be read and its binding state could not be corroborated (the strict floating IP listing failed): %s. Refusing to bind on ambiguous evidence.",
				fipID, cerr,
			)
		}
		switch state {
		case client.FloatingIPBindingBoundToTarget:
			// Already our pair (the read was a transient/permission blip): adopt
			// by confirming + setting, no bind POST.
			return confirmAndSetFloatingIPBinding(ctx, d, fipID, staticID, funcs)
		case client.FloatingIPBindingUnbound:
			// Positively present and UNBOUND: the FIP is provably free.
			// proceed to bind.
		case client.FloatingIPBindingBoundToOther:
			// Positively present and bound to a DIFFERENT static IP: anti-clobber,
			// fail closed with ZERO bind POST (do NOT rely on the API to reject).
			return diag.Errorf(
				"floating IP %s could not be read directly, but the strict listing shows it is already bound to a different static IP; refusing to clobber that binding by binding it to %s. Unbind it first (or import the existing binding).",
				fipID, staticID,
			)
		default:
			return diag.Errorf(
				"floating IP %s could not be read and the strict listing did not positively show it present and unbound; refusing to bind on inconclusive evidence to avoid clobbering an out-of-band binding. Resolve the access error, then retry.",
				fipID,
			)
		}
	}

	// 2. Bind, then wait for the activity (skipped on a 409-idempotent bind) —
	//    with a bounded retry on the transient platform gateway 502 (#315/#319).
	//    The retry wraps ONLY the bind+wait; the pre-bind anti-clobber decision
	//    (above) and the post-bind confirmation (below) stay OUTSIDE it. The
	//    FIRST attempt binds directly (the pre-bind read already proved the FIP
	//    free); a RETRY first re-reads and re-classifies so it never clobbers a
	//    binding that appeared in between, and short-circuits to success once
	//    the bind has converged platform-side.
	firstBindAttempt := true
	bindErr := runVPCWriteWithRetry(ctx, vpcWriteRetry{
		label:       fmt.Sprintf("bind floating IP %s to static IP %s", fipID, staticID),
		sleep:       funcs.retrySleep,
		isTransient: funcs.isTransient,
		attempt: func(ctx context.Context) error {
			if !firstBindAttempt {
				state, serr := classifyFloatingIPBindingLive(ctx, fipID, staticID, funcs)
				if serr != nil {
					return serr
				}
				switch state {
				case client.FloatingIPBindingBoundToTarget:
					// The bind converged platform-side despite the transient
					// failure: success (the external confirmation sets the id).
					return nil
				case client.FloatingIPBindingBoundToOther:
					return fmt.Errorf("floating IP %s became bound to a different static IP between retries; refusing to clobber it by binding it to %s", fipID, staticID)
				case client.FloatingIPBindingInconclusive:
					return fmt.Errorf("floating IP %s has an inconclusive binding state between retries; refusing to re-bind it to %s on ambiguous evidence", fipID, staticID)
				}
				// Unbound: still free, re-bind below.
			}
			firstBindAttempt = false
			activityID, berr := funcs.bind(ctx, fipID, staticID)
			if berr != nil {
				return berr
			}
			return funcs.wait(ctx, activityID)
		},
	})
	if bindErr != nil {
		return diag.Errorf("failed to bind floating IP %s to static IP %s: %s", fipID, staticID, bindErr)
	}

	// 3. Bounded confirmation retry: SetId ONLY after the binding is confirmed
	//    converged in the read model.
	return confirmAndSetFloatingIPBinding(ctx, d, fipID, staticID, funcs)
}

// confirmAndSetFloatingIPBinding polls the per-id read until the floating IP is
// confirmed bound to our static IP, then sets the id and the computed attrs. If
// the bounded budget is exhausted without converging, it returns an error WITHOUT
// setting the id (a subsequent Create's pre-bind read will ADOPT the converged
// pair, so this is a safe recovery — it never orphans the binding).
func confirmAndSetFloatingIPBinding(ctx context.Context, d *schema.ResourceData, fipID, staticID string, funcs vpcFloatingIPBindingFuncs) diag.Diagnostics {
	if funcs.sleep == nil {
		funcs.sleep = defaultFloatingIPBindingSleep
	}
	for attempt := 1; attempt <= maxFloatingIPBindingConfirmAttempts; attempt++ {
		rawFip, err := funcs.read(ctx, fipID)
		if err != nil {
			return diag.Errorf("failed to confirm the binding of floating IP %s to static IP %s: %s", fipID, staticID, err)
		}
		// Convergence counts ONLY on a usable per-id read (carries EXACTLY fipID)
		// that shows our pair. A nil/403/mismatched-id read is not convergence.
		if fip, ok := usableFloatingIPRead(rawFip, fipID); ok && fip.StaticIP != nil && fip.StaticIP.ID == staticID {
			d.SetId(makeFloatingIPBindingID(fipID, staticID))
			writeFloatingIPBindingComputed(d, fip)
			return nil
		}
		if attempt == maxFloatingIPBindingConfirmAttempts {
			break
		}
		if serr := funcs.sleep(ctx, attempt); serr != nil {
			return diag.Errorf("cancelled while confirming the binding of floating IP %s to static IP %s: %s", fipID, staticID, serr)
		}
	}
	return diag.Errorf(
		"floating IP %s was bound to static IP %s but the binding did not converge in the read model within the confirmation budget; the resource id was NOT set to avoid recording an unconfirmed binding. Re-run apply: the next create will adopt the binding once it converges.",
		fipID, staticID,
	)
}

// readVPCFloatingIPBinding holds the testable read logic. State safety is the
// overriding invariant: the resource is NEVER dropped on an inconclusive read.
func readVPCFloatingIPBinding(ctx context.Context, d *schema.ResourceData, fipID, staticID string, funcs vpcFloatingIPBindingFuncs) diag.Diagnostics {
	if funcs.sleep == nil {
		funcs.sleep = defaultFloatingIPBindingSleep
	}

	rawFip, err := funcs.read(ctx, fipID)
	if err != nil {
		return diag.Errorf("failed to read floating IP %s: %s", fipID, err)
	}

	// Only a usable per-id read (carries EXACTLY fipID) can be used as evidence; a
	// mismatched/empty-id body is treated like a nil/ambiguous read and must NEVER
	// trigger SetId("") (#312 R7 guard).
	if fip, ok := usableFloatingIPRead(rawFip, fipID); ok {
		switch client.ClassifyFloatingIPBinding(fip, staticID) {
		case client.FloatingIPBindingBoundToTarget:
			// Present and still our pair: refresh.
			writeFloatingIPBindingComputed(d, fip)
			return nil
		case client.FloatingIPBindingInconclusive:
			// Present, but the nested staticIp is structurally incomplete (a staticIp
			// object with an empty/omitted id). This is NOT proof the pair is gone, so
			// it must NEVER become a stable negative that drops state — treat it like
			// an ambiguous read and keep the binding (#312 R6 applied to the nested id).
			tflog.Warn(ctx, fmt.Sprintf(
				"floating IP %s was read with a structurally incomplete nested staticIp (present but id-less); the binding is kept in the state to avoid a wrong removal on an inconclusive read",
				fipID,
			))
			return nil
		}
		// DEFINITIVE not-our-pair (Unbound, or BoundToOther with a non-empty different
		// id). This could be a stale-200 right after a bind. Do a bounded retry to
		// rule that out; only a STABLE definitive negative drops the resource
		// (positive absence of THIS pair).
		for attempt := 1; attempt < maxFloatingIPBindingConfirmAttempts; attempt++ {
			if serr := funcs.sleep(ctx, attempt); serr != nil {
				return diag.Errorf("cancelled while confirming floating IP %s binding state: %s", fipID, serr)
			}
			rawFip, err = funcs.read(ctx, fipID)
			if err != nil {
				// An ambiguous read mid-retry must not drop: fail closed.
				return diag.Errorf("failed to re-read floating IP %s while confirming its binding state: %s", fipID, err)
			}
			fip, ok = usableFloatingIPRead(rawFip, fipID)
			if !ok {
				// Became ambiguous (nil/403/mismatched-id): do not drop; keep state.
				tflog.Warn(ctx, fmt.Sprintf("floating IP %s became unreadable while confirming its binding state; keeping the binding in state", fipID))
				return nil
			}
			switch client.ClassifyFloatingIPBinding(fip, staticID) {
			case client.FloatingIPBindingBoundToTarget:
				writeFloatingIPBindingComputed(d, fip)
				return nil
			case client.FloatingIPBindingInconclusive:
				// Became inconclusive mid-confirmation (id-less nested staticIp): do
				// not drop; keep state.
				tflog.Warn(ctx, fmt.Sprintf("floating IP %s was re-read with an id-less nested staticIp while confirming its binding state; keeping the binding in state", fipID))
				return nil
			}
		}
		// Stable DEFINITIVE negative: the floating IP is present and provably NOT
		// bound to our static IP (Unbound or BoundToOther). This is positive absence
		// of THIS pair -> drop.
		d.SetId("")
		return nil
	}

	// nil/403-absent or a mismatched-id body -> AMBIGUOUS. Never drop on ambiguity.
	// Keep the state and emit a diagnostic-as-warning so a permission blip or an
	// inconsistent read cannot silently remove the binding.
	tflog.Warn(ctx, fmt.Sprintf(
		"floating IP %s could not be read consistently (absent, access denied, or an id-inconsistent body); the binding is kept in the state to avoid a wrong removal on an ambiguous read",
		fipID,
	))
	return nil
}

// deleteVPCFloatingIPBinding holds the testable delete (unbind) logic. State
// safety: the resource is removed only after STRICT positive confirmation that
// the floating IP is no longer bound to our static IP. A 403 alone is NEVER
// "gone".
func deleteVPCFloatingIPBinding(ctx context.Context, d *schema.ResourceData, fipID, staticID string, funcs vpcFloatingIPBindingFuncs) diag.Diagnostics {
	// Unbind, then wait for the activity — with a bounded retry on the
	// transient platform gateway 502 (#315/#319). The retry wraps ONLY the
	// unbind+wait; the post-unbind confirmation (confirmFloatingIPUnbound)
	// stays OUTSIDE it and rules on the final state. The FIRST attempt unbinds
	// directly; a RETRY first re-reads and re-classifies, and re-issues the
	// unbind ONLY while the pair is still bound to OUR target — otherwise the
	// unbind already took effect (or the pair was never ours) and re-issuing it
	// could disturb a binding that is no longer ours.
	firstUnbindAttempt := true
	unbindErr := runVPCWriteWithRetry(ctx, vpcWriteRetry{
		label:       fmt.Sprintf("unbind floating IP %s from static IP %s", fipID, staticID),
		sleep:       funcs.retrySleep,
		isTransient: funcs.isTransient,
		attempt: func(ctx context.Context) error {
			if !firstUnbindAttempt {
				state, serr := classifyFloatingIPBindingLive(ctx, fipID, staticID, funcs)
				if serr != nil {
					return serr
				}
				if state != client.FloatingIPBindingBoundToTarget {
					// No longer bound to OUR pair (Unbound / BoundToOther /
					// Inconclusive): the unbind took effect (or the pair was
					// never ours). Stop; the external confirmation rules.
					return nil
				}
				// Still our pair: re-issue the unbind below.
			}
			firstUnbindAttempt = false
			activityID, uerr := funcs.unbind(ctx, fipID, staticID)
			if uerr != nil {
				// A 404/403 on the unbind CALL itself is idempotent: the pair
				// is (probably) gone. Stop here and let the strict external
				// confirmation prove it; never retry a permission/absence code.
				if isVPCStatusCode(uerr, http.StatusNotFound) || isVPCStatusCode(uerr, http.StatusForbidden) {
					return nil
				}
				return uerr
			}
			return funcs.wait(ctx, activityID)
		},
	})
	if unbindErr != nil {
		return diag.Errorf("failed to unbind floating IP %s from static IP %s: %s", fipID, staticID, unbindErr)
	}

	return confirmFloatingIPUnbound(ctx, fipID, staticID, funcs, nil)
}

// confirmFloatingIPUnbound accepts the unbind only on strict positive evidence
// that the floating IP is no longer bound to our static IP:
//   - a 200 read showing StaticIP nil or != our static IP (the FIP is
//     pre-provisioned and not deletable here, so the expected post-unbind shape
//     is a present FIP not bound to our pair); OR
//   - a strict-200 listing that positively shows the FIP present and NOT bound to
//     our static IP.
//
// A nil/403 read alone, or an inconclusive listing, FAILS CLOSED. There is NO
// "the FIP is absent from the listing => success" path.
func confirmFloatingIPUnbound(ctx context.Context, fipID, staticID string, funcs vpcFloatingIPBindingFuncs, unbindErr error) diag.Diagnostics {
	rawFip, err := funcs.read(ctx, fipID)
	if err != nil {
		return diag.Errorf("failed to confirm the unbind of floating IP %s from static IP %s: %s", fipID, staticID, err)
	}
	// Only a usable per-id read (carries EXACTLY fipID) can prove the post-unbind
	// shape. A mismatched/empty-id body must NOT be accepted as "gone": fall to the
	// strict-listing corroboration just like a nil/ambiguous read (#312 R7 guard).
	if fip, ok := usableFloatingIPRead(rawFip, fipID); ok {
		switch client.ClassifyFloatingIPBinding(fip, staticID) {
		case client.FloatingIPBindingUnbound, client.FloatingIPBindingBoundToOther:
			// DEFINITIVELY not our pair (staticIp nil, or bound to a different static
			// IP with a non-empty id): positively unbound (or rebound elsewhere).
			return nil
		case client.FloatingIPBindingBoundToTarget:
			// Still bound to our pair: the unbind did not take effect.
			detail := ""
			if unbindErr != nil {
				detail = fmt.Sprintf(" (the unbind call returned %s)", unbindErr)
			}
			return diag.Errorf(
				"floating IP %s is still bound to static IP %s after the unbind; the binding is kept in the state%s. Retry once the unbind succeeds.",
				fipID, staticID, detail,
			)
		}
		// Inconclusive: the nested staticIp is structurally incomplete (present but
		// id-less). This is NOT proof the pair is gone, so it must NOT be accepted as
		// a successful unbind. Fall through to the strict-listing corroboration (and
		// if that is also Inconclusive, the delete FAILS CLOSED below).
	}

	// Read is ambiguous (nil/403/mismatched-id). Corroborate via the strict
	// listing. The unbind is accepted as successful on any state proving the FIP
	// is no longer bound to OUR pair: Unbound (present, staticIp nil) OR
	// BoundToOther (present, bound to a different static IP). BoundToTarget means
	// still our pair (keep state); Inconclusive fails closed (keep state).
	state, cerr := funcs.corroborate(ctx, fipID, staticID)
	if cerr != nil {
		return diag.Errorf(
			"floating IP %s could not be read after the unbind and the strict listing failed (%s); the binding is kept in the state to avoid a wrong removal. Resolve the access error, then retry.",
			fipID, cerr,
		)
	}
	switch state {
	case client.FloatingIPBindingUnbound, client.FloatingIPBindingBoundToOther:
		// No longer bound to OUR pair: the unbind took effect.
		return nil
	case client.FloatingIPBindingBoundToTarget:
		return diag.Errorf(
			"floating IP %s is still bound to static IP %s after the unbind (confirmed by the strict listing); the binding is kept in the state. Retry once the unbind succeeds.",
			fipID, staticID,
		)
	default:
		return diag.Errorf(
			"floating IP %s could not be read after the unbind and the strict listing did not positively show it unbound from static IP %s; the binding is kept in the state to avoid removing it on inconclusive evidence. Resolve the read error, then retry.",
			fipID, staticID,
		)
	}
}

// writeFloatingIPBindingComputed refreshes the read-only address attributes from
// a live floating IP. floating_ip_address comes from the FIP; static_ip_address
// comes from the nested staticIp.address (populated only when bound).
func writeFloatingIPBindingComputed(d *schema.ResourceData, fip *client.FloatingIP) {
	_ = d.Set("floating_ip_address", fip.IPAddress)
	staticAddr := ""
	if fip.StaticIP != nil {
		staticAddr = fip.StaticIP.Address
	}
	_ = d.Set("static_ip_address", staticAddr)
}

func makeFloatingIPBindingID(fipID, staticID string) string {
	return fipID + vpcFloatingIPBindingIDSeparator + staticID
}

// splitFloatingIPBindingID parses "{floatingIpId}:{staticIpId}" into exactly two
// non-empty halves. Anything else (no separator, empty halves, extra
// separators) is an error.
func splitFloatingIPBindingID(id string) (fipID, staticID string, err error) {
	parts := strings.Split(id, vpcFloatingIPBindingIDSeparator)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected exactly two non-empty parts separated by %q", vpcFloatingIPBindingIDSeparator)
	}
	return parts[0], parts[1], nil
}
