package client

import (
	"context"
	"encoding/json"
)

type PublicCloudVMInstanceFamilyClient struct {
	c *Client
}

// InstanceFamily returns the instance-family catalogue sub-client (read-only).
func (v *PublicCloudVMClient) InstanceFamily() *PublicCloudVMInstanceFamilyClient {
	return &PublicCloudVMInstanceFamilyClient{v.c}
}

// PublicCloudVMInstanceFamily mirrors an element of
// GET /vm_instances/v1/instance_families (bare array, camelCase). This is the id
// the VM resource consumes (instance_family_id).
type PublicCloudVMInstanceFamily struct {
	ID          string
	Name        string
	Description string
	VcpuMin     int
	VcpuMax     int
	RamMinGib   int
	RamMaxGib   int
	// Skus is the priced billing catalogue (vCPU and RAM) of the family. The
	// API returns it on both the list and the single-item endpoints (#506).
	// PublicCloudVMSku is the shared SKU type (defined with the storage-type
	// sibling in public_cloud_vm_storage_type.go, #507): the same priced-SKU
	// shape the VM Instances API returns across catalogue resources.
	Skus []PublicCloudVMSku `json:"skus"`
}

// UnmarshalJSON accepts both the current `ramMinGib`/`ramMaxGib` spellings and
// the deprecated `ramMinGb`/`ramMaxGb` ones (see public_cloud_vm_capacity.go,
// issue #524). The explicit `skus` tag on the embedded alias is preserved.
func (f *PublicCloudVMInstanceFamily) UnmarshalJSON(data []byte) error {
	type plain PublicCloudVMInstanceFamily
	var v struct {
		plain
		RamMinGib *int
		RamMinGb  *int
		RamMaxGib *int
		RamMaxGb  *int
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*f = PublicCloudVMInstanceFamily(v.plain)

	subject := "instance family " + quotedOrUnidentified(v.plain.ID)
	ramMin, err := resolveRenamedCapacity(v.RamMinGib, v.RamMinGb, "ramMinGib", "ramMinGb", subject)
	if err != nil {
		return err
	}
	ramMax, err := resolveRenamedCapacity(v.RamMaxGib, v.RamMaxGb, "ramMaxGib", "ramMaxGb", subject)
	if err != nil {
		return err
	}
	f.RamMinGib = ramMin
	f.RamMaxGib = ramMax
	return nil
}

// List returns the instance families of the tenant (bare JSON array, no filter).
func (f *PublicCloudVMInstanceFamilyClient) List(ctx context.Context) ([]*PublicCloudVMInstanceFamily, error) {
	req := f.c.newRequest("GET", "/vm_instances/v1/instance_families")
	resp, err := f.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*PublicCloudVMInstanceFamily
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read returns a single instance family by id. A positive 404 maps to (nil, nil);
// any other non-OK code (403, 5xx) fails closed with an error.
func (f *PublicCloudVMInstanceFamilyClient) Read(ctx context.Context, id string) (*PublicCloudVMInstanceFamily, error) {
	req := f.c.newRequest("GET", "/vm_instances/v1/instance_families/%s", id)
	resp, err := f.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMInstanceFamily
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
