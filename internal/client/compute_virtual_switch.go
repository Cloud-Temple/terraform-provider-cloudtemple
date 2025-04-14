package client

import "context"

type VirtualSwitchClient struct {
	c *Client
}

func (c *ComputeClient) VirtualSwitch() *VirtualSwitchClient {
	return &VirtualSwitchClient{c.c}
}

type VirtualSwitch struct {
	ID             string     `terraform:"id"`
	Name           string     `terraform:"name"`
	Moref          string     `terraform:"moref"`
	FolderID       string     `terraform:"folder_id"`
	MachineManager BaseObject `terraform:"machine_manager"`
}

type VirtualSwitchFilter struct {
	Name             string `filter:"name"`
	MachineManagerId string `filter:"machineManagerId"`
	DatacenterId     string `filter:"datacenterId"`
	HostClusterId    string `filter:"hostClusterId"`
}

func (v *VirtualSwitchClient) List(
	ctx context.Context,
	filter *VirtualSwitchFilter) ([]*VirtualSwitch, error) {

	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_switchs")
	r.addFilter(filter)
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
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_switchs/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out VirtualSwitch
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
