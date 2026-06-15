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

// VPCFloatingIPClient handles read and bind/unbind operations on VPC floating
// IPs. The floating IP itself is provisioned out-of-band (it is NOT created or
// destroyed here); this client only reads it and binds/unbinds it to a static
// IP.
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
// floating IP does not exist (403; the API returns 403 for an absent resource).
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
	// a DIFFERENT static IP. Binding our pair would clobber that out-of-band
	// binding, so this is FAIL-CLOSED for create; for delete it means "no longer
	// bound to OUR pair", i.e. our unbind took effect.
	FloatingIPBindingBoundToOther
)

// CorroborateBinding strictly classifies the FIP/static relationship from a
// COMPLETE HTTP 200 listing. Like vpc_static_ip's ListStrict, it FAILS CLOSED to
// "inconclusive": a listing that cannot prove the floating IP's state (null /
// empty / non-array body, an id-less entry, or the FIP simply not present) is
// NEVER read as negative evidence. Only a positively observed FIP yields a
// definite present-and-(unbound|bound-to-target|bound-to-other) classification.
//
// The four states are kept DISTINCT so the create path can never collapse
// "present & unbound" (safe to bind) with "present & bound to a DIFFERENT static
// IP" (must fail closed, never bind): that collapse was the anti-clobber defect.
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
		if fip.StaticIP == nil {
			// Present and not bound to anything: provably free.
			return FloatingIPBindingUnbound, nil
		}
		if fip.StaticIP.ID == staticID {
			return FloatingIPBindingBoundToTarget, nil
		}
		// Present and bound to a DIFFERENT static IP.
		return FloatingIPBindingBoundToOther, nil
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
