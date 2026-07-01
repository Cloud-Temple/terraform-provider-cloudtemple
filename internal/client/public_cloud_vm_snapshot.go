package client

import (
	"context"
	"fmt"
)

type PublicCloudVMSnapshotClient struct {
	c *Client
}

// Snapshot returns the VM snapshot sub-client. Snapshots are VM-scoped
// (/vm_instances/v1/virtual_machines/{vmID}/snapshots...); create and delete are
// asynchronous (201 + Location:<activityId>). Revert is intentionally not exposed.
func (v *PublicCloudVMClient) Snapshot() *PublicCloudVMSnapshotClient {
	return &PublicCloudVMSnapshotClient{v.c}
}

// PublicCloudVMSnapshot mirrors an element of GET .../snapshots (camelCase,
// verified live). There is no size/description field.
type PublicCloudVMSnapshot struct {
	ID        string
	VmID      string
	TenantID  string
	Name      string
	Status    string
	CreatedAt string
}

// publicCloudVMSnapshotListResponse is the wrapped list shape (verified live):
// {"snapshots": [...], "total": N}. Total is a pointer so a missing `total` field
// is distinguishable from a genuine 0: a malformed/empty wrapper must NOT be
// accepted as authoritative absence evidence.
type publicCloudVMSnapshotListResponse struct {
	Snapshots []*PublicCloudVMSnapshot
	Total     *int
}

// List returns the VM's snapshots (lenient success contract).
func (s *PublicCloudVMSnapshotClient) List(ctx context.Context, vmID string) ([]*PublicCloudVMSnapshot, error) {
	return s.list(ctx, vmID, false)
}

// ListStrict returns the VM's snapshots with a 200-only contract AND a
// completeness check (total == len): the authoritative absence evidence (E0-9).
func (s *PublicCloudVMSnapshotClient) ListStrict(ctx context.Context, vmID string) ([]*PublicCloudVMSnapshot, error) {
	return s.list(ctx, vmID, true)
}

func (s *PublicCloudVMSnapshotClient) list(ctx context.Context, vmID string, strict bool) ([]*PublicCloudVMSnapshot, error) {
	req := s.c.newRequest("GET", "/vm_instances/v1/virtual_machines/%s/snapshots", vmID)
	resp, err := s.c.doRequest(ctx, req)
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

	var out publicCloudVMSnapshotListResponse
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	if strict {
		if out.Total == nil {
			return nil, fmt.Errorf("snapshot listing for virtual machine %s did not report a total; refusing to use a malformed listing as absence evidence", vmID)
		}
		if *out.Total != len(out.Snapshots) {
			return nil, fmt.Errorf("snapshot listing for virtual machine %s is incomplete (total %d, %d returned); refusing to use a truncated listing as absence evidence", vmID, *out.Total, len(out.Snapshots))
		}
	}
	return out.Snapshots, nil
}

// Read returns a single snapshot by id. A positive 404 maps to (nil, nil); any
// other non-OK code (403, 5xx) fails closed with an error.
func (s *PublicCloudVMSnapshotClient) Read(ctx context.Context, vmID, snapshotID string) (*PublicCloudVMSnapshot, error) {
	req := s.c.newRequest("GET", "/vm_instances/v1/virtual_machines/%s/snapshots/%s", vmID, snapshotID)
	resp, err := s.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMSnapshot
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// createVMSnapshotRequest is the body of POST .../snapshots ({"name": "..."} —
// the only accepted field).
type createVMSnapshotRequest struct {
	Name string `json:"name"`
}

// Create takes a snapshot and returns the activityId. The completed activity's
// result / concernedItems ("snapshot") carries the new snapshot id.
func (s *PublicCloudVMSnapshotClient) Create(ctx context.Context, vmID, name string) (string, error) {
	r := s.c.newRequest("POST", "/vm_instances/v1/virtual_machines/%s/snapshots", vmID)
	r.obj = &createVMSnapshotRequest{Name: name}
	return s.c.doRequestAndReturnActivity(ctx, r)
}

// Delete removes a snapshot and returns the activityId (the completed activity's
// result is the vmId, not the snapshotId).
func (s *PublicCloudVMSnapshotClient) Delete(ctx context.Context, vmID, snapshotID string) (string, error) {
	r := s.c.newRequest("DELETE", "/vm_instances/v1/virtual_machines/%s/snapshots/%s", vmID, snapshotID)
	return s.c.doRequestAndReturnActivity(ctx, r)
}
