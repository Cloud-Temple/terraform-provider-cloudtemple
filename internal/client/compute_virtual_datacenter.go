package client

import "context"

type VirtualDatacenterClient struct {
	c *Client
}

func (c *Compute) VirtualDatacenter() *VirtualDatacenterClient {
	return &VirtualDatacenterClient{c.c}
}

type VirtualDatacenter struct {
	ID               string `terraform:"id"`
	Name             string `terraform:"name"`
	MachineManagerID string `terraform:"machine_manager_id"`
	TenantID         string `terraform:"tenant_id"`
}

func (v *VirtualDatacenterClient) List(
	ctx context.Context,
	machineManagerId string,
	name string) ([]*VirtualDatacenter, error) {

	// TODO: filters
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_datacenters")
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*VirtualDatacenter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (v *VirtualDatacenterClient) Read(ctx context.Context, id string) (*VirtualDatacenter, error) {
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_datacenters/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out VirtualDatacenter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
