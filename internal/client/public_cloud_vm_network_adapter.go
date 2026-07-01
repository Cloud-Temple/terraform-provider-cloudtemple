package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type PublicCloudVMNetworkAdapterClient struct {
	c *Client
}

// NetworkAdapter returns the VM network-adapter (NIC) sub-client. NICs are
// VM-scoped (/vm_instances/v1/virtual_machines/{vmID}/network_adapters); every
// write is asynchronous (201 + Location:<activityId>). The broker remaps the
// worker's nic_id to `id`.
func (v *PublicCloudVMClient) NetworkAdapter() *PublicCloudVMNetworkAdapterClient {
	return &PublicCloudVMNetworkAdapterClient{v.c}
}

// PublicCloudVMNetworkAdapter mirrors an element of GET .../network_adapters
// (camelCase, verified live). ipv4Address/ipv6Address arrive as JSON null until
// the guest agent reports them, decoding to "". The spec's `vif_uuid` is NOT
// returned by the live API (neither the list nor the single GET) and is
// intentionally not modelled.
type PublicCloudVMNetworkAdapter struct {
	ID              string
	DeviceIndex     int
	NetworkID       string
	NetworkName     string
	Type            string
	ProvisionStatus string
	MacAddress      string
	IPv4Address     string
	IPv6Address     string
}

// publicCloudVMNetworkAdapterListResponse is the wrapped list shape (verified
// live): {"vmId": "...", "networks": [...], "total": N}. Total is a pointer so a
// missing `total` is distinguishable from a genuine 0 — a malformed wrapper ({}
// or {"networks":[]}) must NOT count as authoritative absence evidence.
type publicCloudVMNetworkAdapterListResponse struct {
	VmID     string
	Networks []*PublicCloudVMNetworkAdapter
	Total    *int
}

// List returns the VM's NICs (lenient success contract).
func (a *PublicCloudVMNetworkAdapterClient) List(ctx context.Context, vmID string) ([]*PublicCloudVMNetworkAdapter, error) {
	return a.list(ctx, vmID, false)
}

// ListStrict returns the VM's NICs with a 200-only contract AND a completeness
// check (total == len(networks)) plus structural integrity checks: a truncated
// page, a wrapper scoped to a different vmId, or a malformed entry can never
// serve as absence evidence. It is the authoritative source for a NIC deletion
// decision (E0-9) — the single-GET code (400 for an unknown NIC) is never trusted
// as absence.
func (a *PublicCloudVMNetworkAdapterClient) ListStrict(ctx context.Context, vmID string) ([]*PublicCloudVMNetworkAdapter, error) {
	return a.list(ctx, vmID, true)
}

func (a *PublicCloudVMNetworkAdapterClient) list(ctx context.Context, vmID string, strict bool) ([]*PublicCloudVMNetworkAdapter, error) {
	req := a.c.newRequest("GET", "/vm_instances/v1/virtual_machines/%s/network_adapters", vmID)
	resp, err := a.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)

	if strict {
		if err := requireHttpCodes(resp, 200); err != nil {
			return nil, err
		}
	} else {
		if err := requireOK(resp); err != nil {
			return nil, err
		}
	}

	var out publicCloudVMNetworkAdapterListResponse
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	if strict {
		if out.Total == nil {
			return nil, fmt.Errorf("network adapter listing for virtual machine %s did not report a total; refusing to use a malformed listing as absence evidence", vmID)
		}
		if *out.Total != len(out.Networks) {
			return nil, fmt.Errorf("network adapter listing for virtual machine %s is incomplete (total %d, %d returned); refusing to use a truncated listing as absence evidence", vmID, *out.Total, len(out.Networks))
		}
		// Structural integrity: a wrapper whose vmId does not match the requested
		// VM, or a nil / empty-id entry, cannot be trusted as a complete-and-correct
		// listing for an absence decision. Compared case-insensitively because an id
		// written upper-case (import/config) must match the platform's lower-case id.
		if out.VmID != "" && !strings.EqualFold(out.VmID, vmID) {
			return nil, fmt.Errorf("network adapter listing reported vmId %q for a request scoped to virtual machine %s; refusing an inconsistent listing", out.VmID, vmID)
		}
		for i, nic := range out.Networks {
			if nic == nil || nic.ID == "" {
				return nil, fmt.Errorf("network adapter listing for virtual machine %s contains a malformed entry at index %d; refusing an untrustworthy listing", vmID, i)
			}
		}
	}
	return out.Networks, nil
}

// Read returns a single NIC by id. A positive 404 maps to (nil, nil); every other
// non-OK code fails closed with an error. NOTE (verified live): an unknown NIC
// returns 400, not 404 — so a caller must NEVER treat a Read error as absence.
// The resource's Read falls back to a complete ListStrict for any drop decision.
func (a *PublicCloudVMNetworkAdapterClient) Read(ctx context.Context, vmID, nicID string) (*PublicCloudVMNetworkAdapter, error) {
	req := a.c.newRequest("GET", "/vm_instances/v1/virtual_machines/%s/network_adapters/%s", vmID, nicID)
	resp, err := a.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	// Explicit contract (stricter than requireNotFoundOrOK, which also accepts
	// 206): ONLY a 200 is a found NIC, ONLY a clean 404 is absence. Everything
	// else — notably the 400 the live API returns for an unknown NIC, and any
	// 206 — fails closed. A caller must never read a non-404 error as absence:
	// the resource confirms a drop via a complete ListStrict, not via this code.
	switch resp.StatusCode {
	case http.StatusOK:
		var out PublicCloudVMNetworkAdapter
		if err := decodeBody(resp, &out); err != nil {
			return nil, err
		}
		return &out, nil
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, generateUnexpectedResponseCodeError(resp)
	}
}

// CreateVMNetworkAdapterRequest is the body of POST .../network_adapters.
// IPAddress is only honoured for VPC networks (the worker registers a static IP
// only when the network type is VPC); it is silently ignored on Private Backbone
// networks, so it is omitted when empty.
type CreateVMNetworkAdapterRequest struct {
	NetworkID   string `json:"networkId"`
	DeviceIndex int    `json:"deviceIndex"`
	IPAddress   string `json:"ipAddress,omitempty"`
}

// Create attaches a NIC and returns the activityId. The completed activity's
// result — and a concernedItem of type network_adapter — carries the new NIC id.
func (a *PublicCloudVMNetworkAdapterClient) Create(ctx context.Context, vmID string, req *CreateVMNetworkAdapterRequest) (string, error) {
	r := a.c.newRequest("POST", "/vm_instances/v1/virtual_machines/%s/network_adapters", vmID)
	r.obj = req
	return a.c.doRequestAndReturnActivity(ctx, r)
}

// ChangeVMNetworkAdapterRequest is the body of PATCH .../{nicID} (change-network).
// Only the target network (and an optional VPC static IP) can change; deviceIndex
// is immutable.
type ChangeVMNetworkAdapterRequest struct {
	NetworkID string `json:"networkId"`
	IPAddress string `json:"ipAddress,omitempty"`
}

// ChangeNetwork re-points a NIC to another network (async). The completed
// activity's result is the vmId (NOT the NIC id, which is unchanged). Requires the
// VM to be stopped — enforced by the worker; a running VM fails the activity.
func (a *PublicCloudVMNetworkAdapterClient) ChangeNetwork(ctx context.Context, vmID, nicID string, req *ChangeVMNetworkAdapterRequest) (string, error) {
	r := a.c.newRequest("PATCH", "/vm_instances/v1/virtual_machines/%s/network_adapters/%s", vmID, nicID)
	r.obj = req
	return a.c.doRequestAndReturnActivity(ctx, r)
}

// Delete detaches a NIC (async) and returns the activityId. The completed
// activity's result is the vmId. Requires the VM to be stopped (enforced by the
// worker; a running VM fails the activity — guarded by the resource beforehand).
func (a *PublicCloudVMNetworkAdapterClient) Delete(ctx context.Context, vmID, nicID string) (string, error) {
	r := a.c.newRequest("DELETE", "/vm_instances/v1/virtual_machines/%s/network_adapters/%s", vmID, nicID)
	return a.c.doRequestAndReturnActivity(ctx, r)
}
