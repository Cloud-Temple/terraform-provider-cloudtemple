package client

import "context"

type VirtualControllerClient struct {
	c *Client
}

func (c *ComputeClient) VirtualController() *VirtualControllerClient {
	return &VirtualControllerClient{c.c}
}

type VirtualController struct {
	ID               string   `terraform:"id"`
	VirtualMachineId string   `terraform:"virtual_machine_id"`
	HotAddRemove     bool     `terraform:"hot_add_remove"`
	Type             string   `terraform:"type"`
	Label            string   `terraform:"label"`
	Summary          string   `terraform:"summary"`
	VirtualDisks     []string `terraform:"virtual_disks"`
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
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*VirtualController
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
