package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// VPCFloatingIPClient handles the FULL lifecycle of VPC floating IPs: provision
// (async create), description update (PATCH), gated deprovision (async delete),
// strict by-id resolution, and bind/unbind to a static IP. A floating IP is a
// BILLABLE public IP, so deprovision is gated by construction (only a provably
// fully-unbound IP is ever deleted) and every destructive decision rests on
// strict positive evidence — see DeprovisionUnbound.
type VPCFloatingIPClient struct {
	c *Client
}

// FloatingIPStaticIP is the static IP a floating IP is bound to, as nested in
// the FloatingIp schema. The API names the address field "address" here
// (distinct from StaticIP.FloatingIP which uses "ipAddress").
type FloatingIPStaticIP struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// FloatingIP represents a floating IP as returned by
// GET /vpc/v1/floating_ips[/{id}].
//
// staticIp, vpc and privateNetwork are nullable in the API: they are populated
// only when the floating IP is bound to a static IP.
type FloatingIP struct {
	ID             string              `json:"id"`
	IPAddress      string              `json:"ipAddress"`
	Description    string              `json:"description"`
	StaticIP       *FloatingIPStaticIP `json:"staticIp"`
	VPC            *BaseObject         `json:"vpc"`
	PrivateNetwork *BaseObject         `json:"privateNetwork"`
}

// FloatingIPFilter narrows a floating-IP listing.
type FloatingIPFilter struct {
	VpcID string `filter:"vpcId"`
}

// List retrieves floating IPs, optionally filtered by VPC ID.
func (f *VPCFloatingIPClient) List(ctx context.Context, filter *FloatingIPFilter) ([]*FloatingIP, error) {
	r := f.c.newRequest("GET", "/vpc/v1/floating_ips")
	r.addFilter(filter)
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*FloatingIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read retrieves a single floating IP by ID. It returns (nil, nil) when the
// floating IP is not found: requireNotFoundOrOK maps BOTH 404 and 403 to not-found
// (the VPC API conflates absent/forbidden, #303), so an absent floating IP —
// whether the API answers 404 or 403 — surfaces as (nil, nil) for idempotent read
// handling.
func (f *VPCFloatingIPClient) Read(ctx context.Context, id string) (*FloatingIP, error) {
	r := f.c.newRequest("GET", "/vpc/v1/floating_ips/%s", id)
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out FloatingIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// ResolveByID is the STRICT by-id read used by the floating-IP RESOURCE (create
// read-back, refresh, delete confirmation) — distinct from Read (datasources),
// which folds 403 into not-found. ResolveByID returns a tri-state result and uses
// an EXPLICIT status switch instead of requireNotFoundOrOK, because for the
// resource only an AUTHORITATIVE 404 may drop state. The floating IP has NO
// listing-omission drop channel (unlike static IP's per-network strict listing),
// so a by-id 404 is the SOLE absence signal:
//
//   - 200         -> decode + id-consistency guard -> (fip, true, nil)
//   - 404         -> (nil, false, nil)  AUTHORITATIVE ABSENT (the sole drop channel)
//   - 403         -> (nil, false, err)  forbidden is NOT absence (#303) — keep + fail
//   - 206 / other -> (nil, false, err)  a partial/unexpected status is never absence
//
// The id-consistency guard fails closed when the 200 body carries an empty id or
// an id that does not match the requested one: a mismatched body must never be
// trusted to represent the requested floating IP.
func (f *VPCFloatingIPClient) ResolveByID(ctx context.Context, id string) (*FloatingIP, bool, error) {
	body, found, err := f.readByID(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	out, err := decodeFloatingIPWithIDGuard(body, id)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// readByID performs the by-id GET and classifies the HTTP status into the strict
// tri-state shared by ResolveByID and the deprovision gate. On 200 it returns the
// RAW body so callers can BOTH decode the struct AND inspect field PRESENCE: an
// omitted association pointer decodes to nil exactly like an explicit null, so
// presence can only be recovered from the raw bytes (see
// floatingIPBodyProvesFullyUnbound). The mapping mirrors ResolveByID's doc:
//
//   - 200         -> (body, true, nil)
//   - 404         -> (nil, false, nil)  AUTHORITATIVE ABSENT (the sole drop channel)
//   - 403         -> (nil, false, err)  forbidden is NOT absence (#303)
//   - 206 / other -> (nil, false, err)  a partial/unexpected status is never absence
func (f *VPCFloatingIPClient) readByID(ctx context.Context, id string) ([]byte, bool, error) {
	r := f.c.newRequest("GET", "/vpc/v1/floating_ips/%s", id)
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, false, err
	}
	defer closeResponseBody(resp)

	switch resp.StatusCode {
	case 200:
		body, rerr := io.ReadAll(resp.Body)
		if rerr != nil {
			return nil, false, rerr
		}
		return body, true, nil
	case 404:
		return nil, false, nil
	case 403:
		return nil, false, fmt.Errorf("floating IP read for %q: access denied (HTTP 403); refusing to treat forbidden as absent (#303)", id)
	default:
		return nil, false, generateUnexpectedResponseCodeError(resp)
	}
}

// decodeFloatingIPWithIDGuard decodes a 200 body into a FloatingIP and enforces the
// id-consistency guard: a body whose id is empty or does not match the requested id
// is never trusted to represent the requested floating IP (fail closed). It runs
// BEFORE any association-evidence check, so a mismatched body's associations are
// never inspected.
func decodeFloatingIPWithIDGuard(body []byte, id string) (*FloatingIP, error) {
	var out FloatingIP
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.ID == "" || out.ID != id {
		return nil, fmt.Errorf("floating IP read for %q returned a body whose id is %q; refusing to trust a mismatched or empty id", id, out.ID)
	}
	return &out, nil
}

// ListStrict retrieves the floating IPs and accepts ONLY a complete,
// structurally valid HTTP 200 array. It mirrors vpc_static_ip's ListStrict: it
// is the corroboration channel for the binding resource, so it must prove
// completeness or FAIL CLOSED — a listing that cannot prove the floating IP's
// state must never be read as negative evidence (the FIP "absent" or "not bound
// to our static IP") (#275/#281 hardening style applied to floating IPs).
//
// Enforced beyond the 200 status:
//   - a partial 206 — or any non-200 (201/403/5xx) — is rejected;
//   - the body MUST be a JSON ARRAY. A 200 whose body is "null", empty, or a
//     JSON object is NOT a list and cannot prove anything (json.Decoder would
//     silently turn "null" into an empty slice);
//   - every entry MUST carry a non-empty id, otherwise the snapshot is
//     structurally incomplete and id-matching against it is unreliable.
//
// No vpcId filter is applied: the corroboration must be able to observe the
// target floating IP wherever it lives.
func (f *VPCFloatingIPClient) ListStrict(ctx context.Context) ([]*FloatingIP, error) {
	r := f.c.newRequest("GET", "/vpc/v1/floating_ips")
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireHttpCodes(resp, 200); err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("strict floating IP listing returned a 200 that is not a JSON array, so it cannot prove the binding state: %.64s", string(trimmed))
	}

	var out []*FloatingIP
	if err := json.Unmarshal(trimmed, &out); err != nil {
		return nil, err
	}
	for i, fip := range out {
		if fip == nil || fip.ID == "" {
			return nil, fmt.Errorf("strict floating IP listing has an entry (index %d) without an id; refusing to use a structurally incomplete listing as corroboration", i)
		}
	}

	return out, nil
}

// Bind associates a PRE-EXISTING floating IP with a static IP (asynchronous).
// The POST returns an activity (Location header); the caller waits for its
// completion. This does NOT create the floating IP — it only establishes the
// FIP -> static IP relationship.
//
// Tolerance for the live API (the vendor swagger is not authoritative):
//   - the call is async (2xx + Location); an empty/sync body is tolerated by
//     doRequestAndReturnActivity, which reads only the Location header;
//   - a 409 Conflict is treated as a non-error idempotent success: the live API
//     may return it when the floating IP is ALREADY bound to the SAME static IP.
//     A 409 returns an empty activity id with a nil error, so the resource layer
//     confirms the actual relationship by a read (a 409 is NOT proof the pair is
//     ours — only the read can establish that). A bind to a DIFFERENT static IP
//     is rejected by the resource layer BEFORE Bind is called: either the
//     pre-bind per-id read, or, when that read is ambiguous, the strict-listing
//     corroboration, which classifies "bound to other" and fails closed.
func (f *VPCFloatingIPClient) Bind(ctx context.Context, fipID, staticID string) (string, error) {
	r := f.c.newRequest("POST", "/vpc/v1/floating_ips/%s/bind/static_ips/%s", fipID, staticID)
	activityID, err := f.c.doRequestAndReturnActivity(ctx, r)
	if err != nil {
		if isVPCConflict(err) {
			// Idempotent: already bound. No activity to wait on; the resource
			// layer confirms the relationship (and that it is OUR pair) by read.
			return "", nil
		}
		return "", err
	}
	return activityID, nil
}

// Unbind removes the association between a floating IP and a static IP
// (asynchronous). The DELETE returns an activity (Location header); the caller
// waits for its completion. The floating IP itself is left intact (it is
// provisioned out-of-band and is not deletable here).
//
// Tolerance mirrors Bind: an empty/sync body is tolerated, and a 409 Conflict is
// treated as a non-error idempotent success (the relationship may already be
// gone). The resource layer confirms the unbind by a read; a 404/403 on the call
// itself is surfaced as an error so the resource layer can apply its strict
// positive-confirmation idempotency rule.
func (f *VPCFloatingIPClient) Unbind(ctx context.Context, fipID, staticID string) (string, error) {
	r := f.c.newRequest("DELETE", "/vpc/v1/floating_ips/%s/unbind/static_ips/%s", fipID, staticID)
	activityID, err := f.c.doRequestAndReturnActivity(ctx, r)
	if err != nil {
		if isVPCConflict(err) {
			return "", nil
		}
		return "", err
	}
	return activityID, nil
}

// ── Floating IP lifecycle: provision / describe / deprovision ──────────────────
//
// These methods make the floating IP a first-class managed resource (C4). A
// floating IP is a BILLABLE public IP with NO client-supplied idempotency key and
// NO create-time tag (the provision body is {"count":1} only), so the create path
// is SPLIT (ProvisionStart + WaitProvision) to let the caller log the activity id
// BEFORE waiting, and deprovision is GATED BY CONSTRUCTION (DeprovisionUnbound).

// provisionFloatingIPRequest is the body of POST /vpc/v1/floating_ips. The live
// contract orders EXACTLY ONE floating IP with {"count":1} and NOTHING else (no
// vpcId, no description — confirmed by live probe). It is unexported because the
// resource never lets the user influence it: the provider always orders one IP.
type provisionFloatingIPRequest struct {
	Count int `json:"count"`
}

// ProvisionStart issues POST /vpc/v1/floating_ips {"count":1} and reports how the
// platform acknowledged the provision WITHOUT waiting. It mirrors the static-IP
// CreateStart split exactly and returns EXACTLY ONE of:
//   - activityID: the ASYNC path — a 201 carrying a Location header; the new
//     floating IP id is resolved once that activity completes (see WaitProvision).
//     This is the deployed live contract (201 + Location, EMPTY body).
//   - syncID: the SYNC path — a 201 carrying a body id. Retained as a DEFENSIVE
//     contract guard so a hypothetical sync 201 surfaces a usable id instead of an
//     orphan; never observed live.
//
// A 201 with NEITHER a Location nor a body id is a hard error (fail closed): the
// provision may have allocated a BILLABLE IP server-side, so the never-orphan
// backstop is the caller's pre-provision diagnostics / manual audit, NOT a silent
// id guess.
//
// Like CreateStart it goes through doRequestWithToken (not doRequest): the live
// async provision returns a Location, so it must bypass the suite-wide
// ErrorOnUnexpectedActivity test guard that doRequest would trip on the EXPECTED
// Location.
func (f *VPCFloatingIPClient) ProvisionStart(ctx context.Context) (activityID, syncID string, err error) {
	r := f.c.newRequest("POST", "/vpc/v1/floating_ips")
	r.obj = provisionFloatingIPRequest{Count: 1}

	token, err := f.c.JWT(ctx)
	if err != nil {
		return "", "", err
	}
	resp, err := f.c.doRequestWithToken(ctx, r, token.Raw)
	if err != nil {
		return "", "", err
	}
	defer closeResponseBody(resp)
	if err := requireHttpCodes(resp, 201); err != nil {
		return "", "", err
	}

	// ASYNC: a Location header is the provision activity to wait on (live contract).
	if loc := resp.Header.Get("Location"); loc != "" {
		return loc, "", nil
	}

	// SYNC fallback (defensive, never observed live): a body id.
	var body struct {
		ID string `json:"id"`
	}
	if derr := decodeBody(resp, &body); derr == nil && body.ID != "" {
		return "", body.ID, nil
	}

	return "", "", fmt.Errorf("floating IP provision returned 201 with neither a Location activity nor an id body; cannot resolve the provisioned id (a billable IP may have been allocated — audit GET /vpc/v1/floating_ips)")
}

// WaitProvision waits for a provision activity and returns the new floating IP id
// from the completed activity's single state Result. It uses the SAME R-M1 core
// as static WaitCreate (waitCreatedIDFromActivity, vpc.go): exactly one state,
// non-empty Result, NO UUID-format check. options controls activity-poll logging.
func (f *VPCFloatingIPClient) WaitProvision(ctx context.Context, activityID string, options *WaiterOptions) (string, error) {
	return f.c.waitCreatedIDFromActivity(ctx, activityID, "floating IP provision", options)
}

// Provision provisions ONE floating IP and returns its id by composing
// ProvisionStart + WaitProvision, EXACTLY like static Create: a SYNC body id (if
// any) short-circuits and is returned directly (NEVER waited on with an empty
// activityID); otherwise it waits on the provision activity. A wait failure is
// wrapped WITH the activityID and manual-recovery text (the IP is billable) and
// NEVER yields (id, nil).
//
// NOTE: the orphan-critical provider Create path does NOT call this composed
// helper — it inlines the split so it can log the activity id BEFORE waiting (see
// resource_vpc_floating_ip.go, C4 phase 3). Provision is the convenience path for
// ct-validate and tests; both honor the same syncID short-circuit.
func (f *VPCFloatingIPClient) Provision(ctx context.Context, options *WaiterOptions) (string, error) {
	activityID, syncID, err := f.ProvisionStart(ctx)
	if err != nil {
		return "", err
	}
	if syncID != "" {
		return syncID, nil
	}
	id, werr := f.WaitProvision(ctx, activityID, options)
	if werr != nil {
		return "", fmt.Errorf("floating IP provision: activity %q did not complete: %w; a billable IP may have been allocated — audit GET /vpc/v1/floating_ips for an unbound, recently-created IP and import-then-destroy or deprovision it", activityID, werr)
	}
	return id, nil
}

// updateFloatingIPDescriptionRequest is the body of PATCH /vpc/v1/floating_ips/{id}.
// It carries ONLY the description (no extra field): the provision body is
// count-only and cannot set a description, so this PATCH is the one channel that
// makes the Terraform-managed description converge. No omitempty — the desired
// value is always non-empty (schema Default + StringIsNotWhiteSpace) and the
// caller always wants the field present.
type updateFloatingIPDescriptionRequest struct {
	Description string `json:"description"`
}

// UpdateDescription issues PATCH /vpc/v1/floating_ips/{id} {"description": value}
// and reports the async handle WITHOUT waiting. Unlike the static-IP PATCH (which
// is async-only and is always waited on), the floating-IP PATCH may be sync OR
// async, so the status is classified FIRST and a Location is read as an async
// handle ONLY on a success status (mirroring CreateStart, which calls
// requireHttpCodes before reading Location):
//
//   - success {200,201,202,204} WITH Location -> (activityID, nil); caller WAITS.
//   - 200/201/204 WITHOUT Location            -> ("", nil) sync success; NO wait.
//   - 202 WITHOUT Location                    -> FAIL CLOSED: a 202 Accepted with no
//     activity handle is async-without-a-handle, so the update cannot be confirmed
//     and must never be reported as success.
//   - 206 / other                             -> error (requireHttpCodes).
//
// Like CreateStart/ProvisionStart it uses doRequestWithToken so a legitimate
// Location bypasses the suite-wide ErrorOnUnexpectedActivity guard. The CALLER
// waits ONLY when the returned activityID is non-empty.
func (f *VPCFloatingIPClient) UpdateDescription(ctx context.Context, fipID, description string) (string, error) {
	r := f.c.newRequest("PATCH", "/vpc/v1/floating_ips/%s", fipID)
	r.obj = updateFloatingIPDescriptionRequest{Description: description}

	token, err := f.c.JWT(ctx)
	if err != nil {
		return "", err
	}
	resp, err := f.c.doRequestWithToken(ctx, r, token.Raw)
	if err != nil {
		return "", err
	}
	defer closeResponseBody(resp)

	// Classify the status FIRST: a Location is an async handle ONLY on success.
	if err := requireHttpCodes(resp, 200, 201, 202, 204); err != nil {
		return "", err
	}
	if loc := resp.Header.Get("Location"); loc != "" {
		return loc, nil
	}
	// No Location: 200/201/204 are sync success; a 202 with no handle is unconfirmable.
	if resp.StatusCode == 202 {
		return "", fmt.Errorf("floating IP %q description PATCH returned 202 Accepted without a Location activity handle; cannot confirm the description update completed", fipID)
	}
	return "", nil
}

// deprovisionRaw issues the async DELETE /vpc/v1/floating_ips/{id} and reports the
// outcome WITHOUT gating or confirming — it is UNEXPORTED on purpose so the ONLY
// reachable deletion path is DeprovisionUnbound, which gates (fully-unbound) and
// confirms (positive 404) around it.
//
// Like ProvisionStart/CreateStart it uses doRequestWithToken so a legitimate
// Location bypasses the suite-wide ErrorOnUnexpectedActivity guard. The status is
// classified FIRST and a Location is read ONLY on a success status:
//
//   - 404                                     -> ("", true, nil)  already gone (idempotent)
//   - success {200,201,202,204} WITH Location -> (activityID, false, nil)  async
//   - success WITHOUT Location                -> ("", false, nil)  sync 2xx
//   - 403                                     -> ("", false, err)  forbidden is NOT deleted (#303)
//   - 206 / other                             -> ("", false, err)  partial/unexpected, never "gone"
func (f *VPCFloatingIPClient) deprovisionRaw(ctx context.Context, fipID string) (activityID string, gone bool, err error) {
	r := f.c.newRequest("DELETE", "/vpc/v1/floating_ips/%s", fipID)

	token, err := f.c.JWT(ctx)
	if err != nil {
		return "", false, err
	}
	resp, err := f.c.doRequestWithToken(ctx, r, token.Raw)
	if err != nil {
		return "", false, err
	}
	defer closeResponseBody(resp)

	switch resp.StatusCode {
	case 404:
		return "", true, nil
	case 200, 201, 202, 204:
		if loc := resp.Header.Get("Location"); loc != "" {
			return loc, false, nil
		}
		return "", false, nil
	case 403:
		return "", false, fmt.Errorf("floating IP deprovision for %q: access denied (HTTP 403); refusing to treat forbidden as deleted (#303)", fipID)
	default:
		return "", false, generateUnexpectedResponseCodeError(resp)
	}
}

// DeprovisionUnbound is the ONLY exported deletion path for a floating IP. It is
// gated AND confirmed BY CONSTRUCTION so a billable IP is deleted ONLY when it is
// provably safe and the deletion is provably complete:
//
//  1. resolveUnboundProof: 404 -> nil (idempotent, the IP is authoritatively
//     absent); error -> error; present -> continue, carrying POSITIVE proof of
//     whether the RAW body shows all three associations PRESENT-and-null.
//  2. Gate on that POSITIVE proof. A clearly-bound IP (non-empty nested staticIp.id)
//     gets a typed refusal; ANY other not-proven-unbound state — a contradictory or
//     id-less association, OR a body that simply OMITS the association fields (an
//     omitted pointer is indistinguishable from null after decode, so absence of the
//     fields is NOT proof of unbound) — FAILS CLOSED with NO DELETE issued
//     ("association evidence inconclusive"). There is no listing-fallback-then-delete:
//     a strict by-id read is sufficient and safer.
//  3. deprovisionRaw: gone -> confirm; activityID -> wait; sync 2xx -> confirm.
//  4. POSITIVE confirmation REQUIRED: a final ResolveByID MUST be 404. A still-200
//     is an error ("deprovision did not remove it"); 403/err fail closed. Success
//     ALWAYS rests on a positive 404, never on a bare 2xx or an empty result.
func (f *VPCFloatingIPClient) DeprovisionUnbound(ctx context.Context, fipID string, options *WaiterOptions) error {
	fip, found, provenUnbound, err := f.resolveUnboundProof(ctx, fipID)
	if err != nil {
		return err
	}
	if !found {
		// Authoritative 404: already gone. Idempotent success on positive evidence.
		return nil
	}

	if !provenUnbound {
		if fip != nil && fip.StaticIP != nil && fip.StaticIP.ID != "" {
			return fmt.Errorf("floating IP %q is bound to static IP %q; unbind or destroy the binding before deprovisioning", fipID, fip.StaticIP.ID)
		}
		return fmt.Errorf("floating IP %q association evidence is inconclusive (staticIp/vpc/privateNetwork are not ALL present-and-null in the read body); refusing to deprovision", fipID)
	}

	activityID, gone, err := f.deprovisionRaw(ctx, fipID)
	if err != nil {
		return err
	}
	if !gone && activityID != "" {
		if _, werr := f.c.Activity().WaitForCompletion(ctx, activityID, options); werr != nil {
			return fmt.Errorf("floating IP %q deprovision activity %q did not complete: %w", fipID, activityID, werr)
		}
	}

	// Positive confirmation: the IP must now be authoritatively absent (404).
	if _, stillPresent, cerr := f.ResolveByID(ctx, fipID); cerr != nil {
		return cerr
	} else if stillPresent {
		return fmt.Errorf("floating IP %q deprovision did not remove it (still present on read-back); refusing to report success", fipID)
	}
	return nil
}

// FloatingIPBindingState classifies, from a strict 200 listing, whether a
// floating IP is present and how it is bound, relative to a target static IP.
type FloatingIPBindingState int

const (
	// FloatingIPBindingInconclusive means the listing could not prove anything
	// about the pair: the listing was not a provably complete 200 array, an
	// entry lacked an id, or the floating IP was simply not seen in the listing.
	// IMPORTANT: a floating IP not seen in a listing is NEVER treated as "the FIP
	// is provably absent" — listing completeness for floating IPs is not proven
	// live, so absence-from-listing is inconclusive, not negative evidence.
	FloatingIPBindingInconclusive FloatingIPBindingState = iota
	// FloatingIPBindingUnbound means the floating IP is present in the listing
	// and is NOT bound to any static IP (staticIp nil). This is the only state
	// that positively proves the FIP is FREE to bind to our target.
	FloatingIPBindingUnbound
	// FloatingIPBindingBoundToTarget means the floating IP is present and bound
	// to the target static IP (positive same-pair corroboration).
	FloatingIPBindingBoundToTarget
	// FloatingIPBindingBoundToOther means the floating IP is present and bound to
	// a DIFFERENT static IP, observed with a NON-EMPTY nested staticIp.id. Binding
	// our pair would clobber that out-of-band binding, so this is FAIL-CLOSED for
	// create; for delete it means "no longer bound to OUR pair", i.e. our unbind
	// took effect. The non-empty id is what makes this a DEFINITIVE not-our-pair:
	// a present-but-id-less nested staticIp is Inconclusive, never BoundToOther.
	FloatingIPBindingBoundToOther
)

// ClassifyFloatingIPBinding classifies a SINGLE, positively observed floating IP
// (a per-id read body or a listing entry) relative to a target static IP id. It
// is the one place the four-way rule lives, so the client (CorroborateBinding)
// and the provider (read stable-negative, delete confirm, create pre-bind) can
// never drift apart on how a nested staticIp is read.
//
// The caller MUST have already established that fip is non-nil and is THE target
// floating IP (its top-level id matches); this helper only inspects the nested
// staticIp relative to staticID:
//
//   - staticIp == nil                          -> Unbound        (definitively free)
//   - staticIp != nil && staticIp.id == ""     -> Inconclusive   (structurally incomplete:
//     the API returned a staticIp OBJECT but omitted its id; this is NOT proof of
//     unbound nor of bound-elsewhere, so it must NEVER be used as negative evidence
//     to drop state or accept an unbind — #312 R6 lesson applied to the nested id)
//   - staticIp != nil && staticIp.id == staticID            -> BoundToTarget (our pair)
//   - staticIp != nil && staticIp.id != "" && != staticID   -> BoundToOther  (definitively not our pair)
func ClassifyFloatingIPBinding(fip *FloatingIP, staticID string) FloatingIPBindingState {
	if fip == nil || fip.StaticIP == nil {
		// staticIp nil: present and bound to nothing -> provably free.
		return FloatingIPBindingUnbound
	}
	if fip.StaticIP.ID == "" {
		// staticIp object present but its id is empty/omitted: structurally
		// incomplete, so we cannot tell unbound from bound-elsewhere. Inconclusive.
		return FloatingIPBindingInconclusive
	}
	if fip.StaticIP.ID == staticID {
		return FloatingIPBindingBoundToTarget
	}
	// Present and bound to a DIFFERENT static IP, with a non-empty id: definitive.
	return FloatingIPBindingBoundToOther
}

// resolveUnboundProof is the DEPROVISION-SPECIFIC gate read. It performs the same
// strict by-id resolution as ResolveByID (same tri-state, same id-consistency
// guard) but ALSO returns whether the 200 body POSITIVELY proves the floating IP is
// fully unbound. The proof is computed from the RAW body, not the decoded struct,
// because an OMITTED association pointer decodes to nil identically to an explicit
// null — so the decoded struct alone cannot tell "the API returned
// staticIp/vpc/privateNetwork as null" from "the API did not return those fields at
// all". A destructive delete must never rest on the latter (a partial/projection
// body), so presence is recovered from the bytes.
func (f *VPCFloatingIPClient) resolveUnboundProof(ctx context.Context, id string) (fip *FloatingIP, found, provenUnbound bool, err error) {
	body, found, err := f.readByID(ctx, id)
	if err != nil {
		return nil, false, false, err
	}
	if !found {
		return nil, false, false, nil
	}
	out, err := decodeFloatingIPWithIDGuard(body, id)
	if err != nil {
		return nil, false, false, err
	}
	return out, true, floatingIPBodyProvesFullyUnbound(body), nil
}

// floatingIPBodyProvesFullyUnbound reports whether the RAW 200 body POSITIVELY
// proves the floating IP is fully unbound: each of "staticIp", "vpc" and
// "privateNetwork" must be PRESENT as a key AND explicitly JSON null. It is the
// DEPROVISION-SPECIFIC unbound proof and is deliberately STRICTER than inspecting
// the decoded struct's nil pointers:
//
//   - an OMITTED field is NOT proof. Go decodes an absent pointer identically to
//     null, so a partial/projection body that simply DROPS the associations would
//     otherwise look "fully unbound" and green-light deleting a BILLABLE,
//     possibly-bound IP. Absence of evidence is not evidence of unbound -> false.
//   - a present-but-non-null association (an object, even an empty "{}") is a
//     binding or a structurally inconclusive shape, never proof of unbound -> false.
//   - the contract populates staticIp/vpc/privateNetwork TOGETHER when bound, so a
//     staticIp:null with a NON-null vpc/privateNetwork is a CONTRADICTION -> false.
//   - a body with DUPLICATE top-level keys is structurally untrustworthy and is
//     REJECTED. Go's json decode silently keeps the LAST value for a duplicate key,
//     so a contradictory body like {"staticIp":{"id":"si-1"},"staticIp":null,...}
//     would decode to staticIp:null and FALSELY look unbound. The proof walks the RAW
//     token stream to reject ANY duplicate top-level key before trusting the map.
//
// This is deliberately DISTINCT from ClassifyFloatingIPBinding / CorroborateBinding,
// which key ONLY on staticIp to answer the C5 binding resource's anti-clobber
// question ("is this FIP free to bind to MY static IP, or bound elsewhere?") — a
// different question from "is it safe to DELETE the FIP entirely?". Folding the two
// together would let one question's "good enough" leak into the other's destructive
// decision.
func floatingIPBodyProvesFullyUnbound(body []byte) bool {
	// A body with duplicate top-level keys cannot be trusted: Go's json decode keeps
	// the LAST value, so {"staticIp":{"id":"si-1"},"staticIp":null,...} would decode to
	// a null staticIp and falsely look unbound. Reject it before reading the map.
	if hasDuplicateTopLevelKeys(body) {
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return false
	}
	for _, key := range []string{"staticIp", "vpc", "privateNetwork"} {
		raw, present := fields[key]
		if !present || string(bytes.TrimSpace(raw)) != "null" {
			return false
		}
	}
	return true
}

// hasDuplicateTopLevelKeys reports whether the JSON object in body repeats any
// top-level key. It walks the RAW token stream because json.Unmarshal into a map
// silently collapses duplicates to the LAST value, which would hide a contradiction
// from the destructive gate. A non-object body or any parse error yields false here:
// those are not "a duplicate" and are handled by the caller's strict json.Unmarshal,
// whose job is the structural-validity check. ONLY a positively observed duplicate
// top-level key returns true. Nested duplicates are irrelevant: a non-null association
// already fails the unbound proof, and there is nothing to nest inside a null.
func hasDuplicateTopLevelKeys(body []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(body))
	tok, err := dec.Token()
	if err != nil {
		return false
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return false
	}
	seen := make(map[string]struct{})
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return false
		}
		key, ok := keyTok.(string)
		if !ok {
			return false
		}
		if _, dup := seen[key]; dup {
			return true
		}
		seen[key] = struct{}{}
		// Consume (and discard) the value so the decoder advances to the next key.
		if err := dec.Decode(new(json.RawMessage)); err != nil {
			return false
		}
	}
	return false
}

// CorroborateBinding strictly classifies the FIP/static relationship from a
// COMPLETE HTTP 200 listing. Like vpc_static_ip's ListStrict, it FAILS CLOSED to
// "inconclusive": a listing that cannot prove the floating IP's state (null /
// empty / non-array body, an id-less entry, or the FIP simply not present) is
// NEVER read as negative evidence. Even when the FIP IS observed, the nested
// staticIp can itself be structurally incomplete (a staticIp object present but
// with an empty/omitted id): that too is Inconclusive, never negative evidence.
// Only a structurally COMPLETE observation yields a definite present-and-
// (unbound|bound-to-target|bound-to-other) classification.
//
// The states are kept DISTINCT so the create path can never collapse
// "present & unbound" (safe to bind) with "present & bound to a DIFFERENT static
// IP" (must fail closed, never bind): that collapse was the anti-clobber defect.
// The classification itself is delegated to ClassifyFloatingIPBinding so the rule
// (including the id-less-nested-staticIp -> Inconclusive case) cannot drift
// between the client and the provider paths.
//
// The "id" the live API returns inside FloatingIP.StaticIP is the static IP id
// (FloatingIPStaticIP.ID), so the same-pair test compares fip.StaticIP.ID to the
// target static IP id.
func (f *VPCFloatingIPClient) CorroborateBinding(ctx context.Context, fipID, staticID string) (FloatingIPBindingState, error) {
	list, err := f.ListStrict(ctx)
	if err != nil {
		return FloatingIPBindingInconclusive, err
	}
	for _, fip := range list {
		if fip == nil || fip.ID != fipID {
			continue
		}
		// The FIP is positively observed in a structurally complete listing. Defer
		// to the shared four-way classifier: a present-but-id-less nested staticIp
		// is Inconclusive (structurally incomplete), NOT BoundToOther — so it can
		// never be used as negative evidence by the resource layer (#312 R6 applied
		// to the nested staticIp.id).
		return ClassifyFloatingIPBinding(fip, staticID), nil
	}
	// The floating IP was not observed in the listing. Listing completeness is
	// NOT proven for floating IPs, so this is inconclusive, never "absent".
	return FloatingIPBindingInconclusive, nil
}

// isVPCConflict reports whether err is a 409 Conflict StatusError.
func isVPCConflict(err error) bool {
	var statusErr StatusError
	return errors.As(err, &statusErr) && statusErr.Code == http.StatusConflict
}
