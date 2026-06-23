package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// VPCStaticIPClient handles read operations on VPC static IPs.
type VPCStaticIPClient struct {
	c *Client
}

// StaticIPFloatingIP is the floating IP bound to a static IP, as nested in the
// StaticIp schema. It carries the floating IP id and address; the API names the
// address field "ipAddress" here (distinct from FloatingIP.StaticIP which uses
// "address").
type StaticIPFloatingIP struct {
	ID        string `json:"id"`
	IPAddress string `json:"ipAddress"`
}

// StaticIP represents a static IP as returned by GET /vpc/v1/static_ips/{id},
// GET /vpc/v1/static_ips/mac/{mac} and
// GET /vpc/v1/private_networks/{id}/static_ips.
//
// virtualMachine, networkAdapter, resourceDescription and floatingIp are
// nullable in the API. source is one of xoa, vmware or custom.
type StaticIP struct {
	ID                  string              `json:"id"`
	IPAddress           string              `json:"ipAddress"`
	MacAddress          string              `json:"macAddress"`
	VirtualMachine      *BaseObject         `json:"virtualMachine"`
	NetworkAdapter      *BaseObject         `json:"networkAdapter"`
	Source              string              `json:"source"`
	ResourceDescription *string             `json:"resourceDescription"`
	FloatingIP          *StaticIPFloatingIP `json:"floatingIp"`
	VPC                 BaseObject          `json:"vpc"`
	PrivateNetwork      BaseObject          `json:"privateNetwork"`
}

// StaticIPFilter narrows a per-private-network static IP listing.
type StaticIPFilter struct {
	VirtualMachineID string `filter:"virtualMachineId"`
}

// List retrieves the static IP mappings of a private network, optionally
// filtered by virtual machine ID.
func (s *VPCStaticIPClient) List(ctx context.Context, privateNetworkID string, filter *StaticIPFilter) ([]*StaticIP, error) {
	r := s.c.newRequest("GET", "/vpc/v1/private_networks/%s/static_ips", privateNetworkID)
	r.addFilter(filter)
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*StaticIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// ListStrict retrieves the static IP mappings of a private network and accepts
// ONLY a complete, structurally valid HTTP 200 listing. Unlike List (which uses
// requireOK and would accept a 206 partial page), ListStrict is the
// deletion-evidence channel: it is what lets Read confirm a deletion and what
// lets Delete confirm a 403-absence, so it must prove completeness or FAIL
// CLOSED — a listing that cannot prove "every static IP on this network" must
// never be read as "the static IP is not present" (#275/#281).
//
// "Provably complete" is enforced beyond the 200 status:
//   - a partial 206 — or any non-200 (201/403/5xx) — is rejected;
//   - the body MUST be a JSON ARRAY. A 200 whose body is "null", empty, or a
//     JSON object is NOT a list and cannot prove absence (json.Decoder would
//     silently turn "null" into an empty slice, i.e. a false "empty network");
//   - every entry MUST carry a non-empty id. An entry without an id means the
//     response is structurally incomplete, so id-matching against it is
//     unreliable and the listing cannot be trusted as evidence.
//
// No VirtualMachineID filter is applied: the confirmation must see every static
// IP on the network, not a VM-scoped subset.
func (s *VPCStaticIPClient) ListStrict(ctx context.Context, privateNetworkID string) ([]*StaticIP, error) {
	r := s.c.newRequest("GET", "/vpc/v1/private_networks/%s/static_ips", privateNetworkID)
	resp, err := s.c.doRequest(ctx, r)
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
	// The body must be a real JSON array. Reject "null"/empty/object outright:
	// they are not a provable listing and must not be read as an empty network.
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("strict static IP listing of private network %s returned a 200 that is not a JSON array, so it cannot prove completeness: %.64s", privateNetworkID, string(trimmed))
	}

	var out []*StaticIP
	if err := json.Unmarshal(trimmed, &out); err != nil {
		return nil, err
	}
	// A complete listing has a usable id for every entry. A missing id makes the
	// snapshot structurally incomplete, so it cannot prove presence/absence.
	for i, si := range out {
		if si == nil || si.ID == "" {
			return nil, fmt.Errorf("strict static IP listing of private network %s has an entry (index %d) without an id; refusing to use a structurally incomplete listing as deletion evidence", privateNetworkID, i)
		}
	}

	return out, nil
}

// Read retrieves a single static IP by ID. It returns (nil, nil) when the static
// IP is not found: requireNotFoundOrOK maps BOTH 404 and 403 to not-found (the VPC
// API conflates absent/forbidden, #303), so an absent static IP — whether the API
// answers 404 or 403 — surfaces as (nil, nil) for idempotent read handling.
func (s *VPCStaticIPClient) Read(ctx context.Context, id string) (*StaticIP, error) {
	r := s.c.newRequest("GET", "/vpc/v1/static_ips/%s", id)
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out StaticIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// CreateStaticIPRequest is the body of POST
// /vpc/v1/private_networks/{privateNetworkId}/static_ips (schema CreateStaticIp).
// MacAddress is required; IPAddress is optional (auto-assigned when omitted, so it
// keeps omitempty).
//
// ResourceDescription is REQUIRED by the live create contract: an empty or omitted
// value is rejected by the deployed API (live evidence, 2026-06 — the swagger's
// "optional" is wrong for this field). It therefore carries NO omitempty: an empty
// string must surface as a CreateStart precondition error (an actionable diagnostic
// BEFORE the POST), never be silently elided from the body and bounced server-side.
type CreateStaticIPRequest struct {
	MacAddress          string `json:"macAddress"`
	IPAddress           string `json:"ipAddress,omitempty"`
	ResourceDescription string `json:"resourceDescription"`
}

// CreateStart issues POST /vpc/v1/private_networks/{id}/static_ips and reports how
// the platform acknowledged the create WITHOUT waiting. It returns EXACTLY ONE of:
//   - activityID: the ASYNC path — a 201 carrying a Location header; the new static
//     IP id is resolved once that activity completes (see WaitCreate). This is the
//     deployed live contract (2026-06: 201 + Location, EMPTY body).
//   - syncID: the SYNC path — a 201 carrying a body static_ip_id. Retained as a
//     DEFENSIVE contract guard so a hypothetical sync 201 surfaces a usable id
//     instead of an orphan; never observed live.
//
// A 201 with NEITHER a Location nor a static_ip_id is a hard error (fail closed):
// the create may have taken effect server-side, so the never-orphan backstop is the
// pre-create teardown net (cmd/ct-validate), NOT a silent id guess. This removes the
// old "resolve the id by listing the network and matching MAC+custom" path, which
// was orphan-prone (it could bind to a co-resident xoa IP, #311) and is moot now
// that the contract is proven async.
//
// ResourceDescription is REQUIRED (see CreateStaticIPRequest): an empty/whitespace
// value is rejected HERE, before the POST, with an actionable error — a doomed
// request never reaches the API.
func (s *VPCStaticIPClient) CreateStart(ctx context.Context, privateNetworkID string, req *CreateStaticIPRequest) (activityID, syncID string, err error) {
	if strings.TrimSpace(req.ResourceDescription) == "" {
		return "", "", fmt.Errorf("static IP create on private network %s: resourceDescription is required by the API and must not be empty or whitespace", privateNetworkID)
	}

	r := s.c.newRequest("POST", "/vpc/v1/private_networks/%s/static_ips", privateNetworkID)
	r.obj = req
	// CreateStart EXPECTS an activity (the live async create returns a Location), so it
	// must bypass the ErrorOnUnexpectedActivity guard the same way the sibling async
	// methods Update/Delete do — they go through doRequestAndReturnActivity, which
	// calls doRequestWithToken directly. Routing through doRequest/doRequestOnce would
	// instead let that guard (set suite-wide by internal/client/tests) reject the
	// EXPECTED Location before line 204 can read it. CreateStart cannot reuse
	// doRequestAndReturnActivity wholesale because it must ALSO accept the defensive
	// sync path (201 + body static_ip_id, no Location), which that helper rejects.
	token, err := s.c.JWT(ctx)
	if err != nil {
		return "", "", err
	}
	resp, err := s.c.doRequestWithToken(ctx, r, token.Raw)
	if err != nil {
		return "", "", err
	}
	defer closeResponseBody(resp)
	if err := requireHttpCodes(resp, 201); err != nil {
		return "", "", err
	}

	// ASYNC: a Location header is the create activity to wait on (live contract).
	if loc := resp.Header.Get("Location"); loc != "" {
		return loc, "", nil
	}

	// SYNC fallback (defensive): a body static_ip_id. Decoded inline — the old named
	// createStaticIPResponse type retired with the sync-create design.
	var body struct {
		StaticIPID string `json:"static_ip_id"`
	}
	if derr := decodeBody(resp, &body); derr == nil && body.StaticIPID != "" {
		return "", body.StaticIPID, nil
	}

	return "", "", fmt.Errorf("static IP create on private network %s returned 201 with neither a Location activity nor a static_ip_id body; cannot resolve the created id", privateNetworkID)
}

// WaitCreate waits for a create activity and returns the new static IP id from the
// completed activity's single state Result — the same channel the provider reads
// via setIdFromActivityState. It fails closed when the activity does not complete
// with EXACTLY ONE state, or completes with an EMPTY Result (R-M1): a created id we
// cannot read must be an error, never an empty id that would orphan the resource
// via SetId(""). options controls activity-poll logging (caller-supplied, like
// every other waiter in this client).
func (s *VPCStaticIPClient) WaitCreate(ctx context.Context, activityID string, options *WaiterOptions) (string, error) {
	act, err := s.c.Activity().WaitForCompletion(ctx, activityID, options)
	if err != nil {
		return "", err
	}
	if act == nil || len(act.State) != 1 {
		return "", fmt.Errorf("static IP create activity %q did not complete with exactly one state; cannot resolve the created id", activityID)
	}
	var id string
	for _, st := range act.State {
		id = st.Result
	}
	if id == "" {
		return "", fmt.Errorf("static IP create activity %q completed with an empty Result; cannot resolve the created id", activityID)
	}
	return id, nil
}

// Create creates a static IP mapping on a private network and returns the new id by
// composing CreateStart + WaitCreate: a SYNC body id (if any) is returned directly;
// otherwise it waits on the create activity. A wait failure is wrapped WITH the
// activityID (so a caller/postmortem can correlate the orphan window) and NEVER
// yields (id, nil) — the provider must fail closed and not SetId (R-Q2).
func (s *VPCStaticIPClient) Create(ctx context.Context, privateNetworkID string, req *CreateStaticIPRequest, options *WaiterOptions) (string, error) {
	activityID, syncID, err := s.CreateStart(ctx, privateNetworkID, req)
	if err != nil {
		return "", err
	}
	if syncID != "" {
		return syncID, nil
	}
	id, werr := s.WaitCreate(ctx, activityID, options)
	if werr != nil {
		return "", fmt.Errorf("static IP create on private network %s: activity %q did not complete: %w", privateNetworkID, activityID, werr)
	}
	return id, nil
}

// UpdateStaticIPRequest is the body of PATCH /vpc/v1/static_ips/{id} (schema
// UpdateStaticIpPayload). The real payload contains ONLY resourceDescription and
// macAddress (verified against the swagger): there is deliberately no ipAddress
// here, because the address is auto-assigned at creation and is not updatable.
// Both fields are omitempty (the schema requires minProperties: 1) so only the
// changed fields are sent in a diff-driven PATCH.
type UpdateStaticIPRequest struct {
	ResourceDescription *string `json:"resourceDescription,omitempty"`
	MacAddress          *string `json:"macAddress,omitempty"`
}

// Update patches a static IP (asynchronous). The PATCH returns an activity
// (Location header); the caller waits for its completion. Only the changed
// updatable fields should be set on req.
func (s *VPCStaticIPClient) Update(ctx context.Context, id string, req *UpdateStaticIPRequest) (string, error) {
	r := s.c.newRequest("PATCH", "/vpc/v1/static_ips/%s", id)
	r.obj = req
	return s.c.doRequestAndReturnActivity(ctx, r)
}

// Delete deletes a static IP (asynchronous). The DELETE returns an activity
// (Location header); the caller waits for its completion.
func (s *VPCStaticIPClient) Delete(ctx context.Context, id string) (string, error) {
	r := s.c.newRequest("DELETE", "/vpc/v1/static_ips/%s", id)
	return s.c.doRequestAndReturnActivity(ctx, r)
}

// ReadByMAC retrieves a single static IP by MAC address. It returns (nil, nil)
// when no static IP matches: requireNotFoundOrOK maps BOTH 404 and 403 to not-found
// (the VPC API conflates absent/forbidden, #303).
func (s *VPCStaticIPClient) ReadByMAC(ctx context.Context, mac string) (*StaticIP, error) {
	r := s.c.newRequest("GET", "/vpc/v1/static_ips/mac/%s", mac)
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out StaticIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
