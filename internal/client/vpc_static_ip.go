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

// Read retrieves a single static IP by ID. It returns (nil, nil) when the
// static IP does not exist (403; the API returns 403 for an absent resource).
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
// MacAddress is required; IPAddress is optional (auto-assigned when omitted);
// ResourceDescription is optional. Optional fields are omitempty so an empty
// value is not sent as an explicit "".
type CreateStaticIPRequest struct {
	MacAddress          string `json:"macAddress"`
	IPAddress           string `json:"ipAddress,omitempty"`
	ResourceDescription string `json:"resourceDescription,omitempty"`
}

// createStaticIPResponse decodes the 201 body of the create endpoint. The static
// IP id is returned in the BODY (static_ip_id), NOT in a Location header: the
// create is synchronous and is NOT an activity, so doRequestAndReturnActivity
// (which reads Location) must not be used here.
type createStaticIPResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	StaticIPID string `json:"static_ip_id"`
}

// Create creates a static IP mapping on a private network (synchronous). It
// returns the new static IP id parsed from the 201 body. A non-201 status, or a
// 201 with a missing/empty static_ip_id, is an error.
func (s *VPCStaticIPClient) Create(ctx context.Context, privateNetworkID string, req *CreateStaticIPRequest) (string, error) {
	r := s.c.newRequest("POST", "/vpc/v1/private_networks/%s/static_ips", privateNetworkID)
	r.obj = req
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return "", err
	}
	defer closeResponseBody(resp)
	if err := requireHttpCodes(resp, 201); err != nil {
		return "", err
	}

	// The swagger documents a 201 body {success, message, static_ip_id}, but the
	// live API returns 201 with an EMPTY body. The create is synchronous (the
	// static IP exists immediately), so when the id is absent from the body we
	// resolve it ourselves. Returning an error on an empty body would be wrong:
	// the static IP IS created, so the caller would orphan it — created
	// platform-side but absent from the Terraform state.
	var out createStaticIPResponse
	if err := decodeBody(resp, &out); err == nil && out.StaticIPID != "" {
		return out.StaticIPID, nil
	}

	// Resolve the id from a COMPLETE listing of the private network, matching the
	// requested MAC AND source=="custom". A by-MAC GET is NOT enough: when a
	// network adapter is attached to a VPC network the platform auto-creates an
	// "xoa" static IP for the SAME MAC (#311), so a MAC-only lookup can resolve to
	// that co-resident platform-managed IP instead of the custom one we just
	// created — which would bind the state to the wrong id and orphan ours. The
	// per-network listing lets us pick the custom IP unambiguously, and the create
	// always produces a "custom" static IP.
	list, lerr := s.ListStrict(ctx, privateNetworkID)
	if lerr != nil {
		return "", fmt.Errorf("static IP created on private network %s but resolving its id by listing failed: %w", privateNetworkID, lerr)
	}
	want := normalizeMAC(req.MacAddress)
	var match *StaticIP
	for _, si := range list {
		if si == nil || si.Source != "custom" || normalizeMAC(si.MacAddress) != want {
			continue
		}
		if match != nil {
			return "", fmt.Errorf("static IP created on private network %s but its id is ambiguous: several custom static IPs carry MAC %s", privateNetworkID, req.MacAddress)
		}
		match = si
	}
	// match.ID == "" must fail just like match == nil: returning an empty id with a
	// nil error would make the resource SetId("") and orphan the created static IP.
	if match == nil || match.ID == "" {
		return "", fmt.Errorf("static IP created on private network %s but could not be resolved among the network's custom static IPs by MAC %s", privateNetworkID, req.MacAddress)
	}
	return match.ID, nil
}

// normalizeMAC canonicalises a MAC address for comparison: lowercase and
// ":"-separated (the swagger pattern also tolerates "-"). It lets the created
// static IP be matched against the requested MAC regardless of input formatting.
func normalizeMAC(mac string) string {
	return strings.ToLower(strings.ReplaceAll(mac, "-", ":"))
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
// when no static IP matches the MAC (403; the API returns 403 for an absent resource).
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
