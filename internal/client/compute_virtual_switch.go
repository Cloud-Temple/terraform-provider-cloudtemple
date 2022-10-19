package client

import "context"

type VirtualSwitchClient struct {
	c *Client
}

func (c *Compute) VirtualSwitch() *VirtualSwitchClient {
	return &VirtualSwitchClient{c.c}
}

type VirtualSwitch struct {
	ID               string
	Name             string
	Moref            string
	FolderID         string
	MachineManagerID string
}

func (v *VirtualSwitchClient) List(
	ctx context.Context,
	machineManagerId string,
	virtualDatacenterId string,
	hostClusterId string) ([]*VirtualSwitch, error) {

	// TODO: filters
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_switchs")
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*VirtualSwitch
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (v *VirtualSwitchClient) Read(ctx context.Context, id string) (*VirtualSwitch, error) {
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_switchs/"+id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out VirtualSwitch
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
