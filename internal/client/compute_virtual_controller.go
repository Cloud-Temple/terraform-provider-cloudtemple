package client

import "context"

type VirtualControllerClient struct {
	c *Client
}

func (c *Compute) VirtualController() *VirtualControllerClient {
	return &VirtualControllerClient{c.c}
}

type VirtualController struct {
	ID               string
	VirtualMachineId string
	HotAddRemove     bool
	Type             string
	Label            string
	Summary          string
	VirtualDisks     []string
}

func (v *VirtualControllerClient) List(
	ctx context.Context,
	virtualMachineId string,
	types string) ([]*VirtualController, error) {

	// TODO: filters
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_controllers")
	r.params.Add("virtualMachineId", virtualMachineId)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*VirtualController
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
