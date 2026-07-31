package client

import (
	"context"
	"fmt"
)

type PublicCloudVMDiskClient struct {
	c *Client
}

// Disk returns the VM disk sub-client. Disks are VM-scoped
// (/vm_instances/v1/virtual_machines/{vmID}/disks...); every write is
// asynchronous (201 + Location:<activityId>). This resource layer manages DATA
// disks only — the system/primary disk is provided by the image and cannot be
// created or deleted here.
func (v *PublicCloudVMClient) Disk() *PublicCloudVMDiskClient {
	return &PublicCloudVMDiskClient{v.c}
}

// PublicCloudVMDisk mirrors an element of GET .../disks (camelCase, verified
// live). Position 0 / IsPrimary true is the system disk; position >= 1 are data
// disks. StorageType is a storage-type UUID.
type PublicCloudVMDisk struct {
	ID          string
	Position    int
	Label       string
	SizeGb      int
	StorageType string
	IsPrimary   bool
}

// publicCloudVMDiskListResponse is the wrapped list shape (verified live):
// {"vmId": "...", "disks": [...], "total": N}. Total is a pointer so a missing
// `total` field is distinguishable from a genuine 0: a malformed/empty wrapper
// ({} or {"disks":[]}) must NOT be accepted as authoritative absence evidence.
type publicCloudVMDiskListResponse struct {
	VmID  string
	Disks []*PublicCloudVMDisk
	Total *int
}

// List returns the VM's disks (lenient success contract).
func (d *PublicCloudVMDiskClient) List(ctx context.Context, vmID string) ([]*PublicCloudVMDisk, error) {
	return d.list(ctx, vmID, false)
}

// ListStrict returns the VM's disks with a 200-only contract AND a completeness
// check (total == len(disks)): a truncated page can never serve as absence
// evidence. It is the authoritative source for a disk deletion decision (E0-9).
func (d *PublicCloudVMDiskClient) ListStrict(ctx context.Context, vmID string) ([]*PublicCloudVMDisk, error) {
	return d.list(ctx, vmID, true)
}

func (d *PublicCloudVMDiskClient) list(ctx context.Context, vmID string, strict bool) ([]*PublicCloudVMDisk, error) {
	req := d.c.newRequest("GET", "/vm_instances/v1/virtual_machines/%s/disks", vmID)
	resp, err := d.c.doRequest(ctx, req)
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

	var out publicCloudVMDiskListResponse
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	if strict {
		if out.Total == nil {
			return nil, fmt.Errorf("disk listing for virtual machine %s did not report a total; refusing to use a malformed listing as absence evidence", vmID)
		}
		if *out.Total != len(out.Disks) {
			return nil, fmt.Errorf("disk listing for virtual machine %s is incomplete (total %d, %d returned); refusing to use a truncated listing as absence evidence", vmID, *out.Total, len(out.Disks))
		}
	}
	return out.Disks, nil
}

// Read returns a single disk by id. A positive 404 (disk OR its VM absent) maps
// to (nil, nil); any other non-OK code (403, 5xx) fails closed with an error.
func (d *PublicCloudVMDiskClient) Read(ctx context.Context, vmID, diskID string) (*PublicCloudVMDisk, error) {
	req := d.c.newRequest("GET", "/vm_instances/v1/virtual_machines/%s/disks/%s", vmID, diskID)
	resp, err := d.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMDisk
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateVMDiskRequest is the body of POST .../disks. Only Size is required;
// StorageType (a storage-type UUID) and Name default server-side when omitted.
// This endpoint only ever adds a DATA disk.
type CreateVMDiskRequest struct {
	Size        int    `json:"size"`
	StorageType string `json:"storageType,omitempty"`
	Name        string `json:"name,omitempty"`
}

// Create adds a data disk and returns the activityId. The completed activity's
// result carries the new disk id.
func (d *PublicCloudVMDiskClient) Create(ctx context.Context, vmID string, req *CreateVMDiskRequest) (string, error) {
	r := d.c.newRequest("POST", "/vm_instances/v1/virtual_machines/%s/disks", vmID)
	r.obj = req
	return d.c.doRequestAndReturnActivity(ctx, r)
}

// extendVMDiskRequest is the body of the extend endpoints ({"size": N}).
type extendVMDiskRequest struct {
	Size int `json:"size"`
}

// ExtendById grows a specific disk (grow-only, and the VM must be stopped — both
// enforced downstream; a violation surfaces as a failed activity). Returns the
// activityId.
func (d *PublicCloudVMDiskClient) ExtendById(ctx context.Context, vmID, diskID string, size int) (string, error) {
	r := d.c.newRequest("POST", "/vm_instances/v1/virtual_machines/%s/disks/%s/extend", vmID, diskID)
	r.obj = &extendVMDiskRequest{Size: size}
	return d.c.doRequestAndReturnActivity(ctx, r)
}

// Delete removes a data disk and returns the activityId. Deleting the primary
// disk is refused downstream (and guarded by the resource before calling this).
func (d *PublicCloudVMDiskClient) Delete(ctx context.Context, vmID, diskID string) (string, error) {
	r := d.c.newRequest("DELETE", "/vm_instances/v1/virtual_machines/%s/disks/%s", vmID, diskID)
	return d.c.doRequestAndReturnActivity(ctx, r)
}

// ExtendSystem grows the VM's system (primary) disk via the dedicated endpoint
// (POST /disks/extend, no diskId — the worker targets the primary disk). Like a
// by-id extend it is grow-only and requires the VM to be stopped (enforced
// downstream). Used by the VM resource's os_disk, not by the data-disk resource.
func (d *PublicCloudVMDiskClient) ExtendSystem(ctx context.Context, vmID string, size int) (string, error) {
	r := d.c.newRequest("POST", "/vm_instances/v1/virtual_machines/%s/disks/extend", vmID)
	r.obj = &extendVMDiskRequest{Size: size}
	return d.c.doRequestAndReturnActivity(ctx, r)
}
