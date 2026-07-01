package client

import "context"

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
	RamMinGb    int
	RamMaxGb    int
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
