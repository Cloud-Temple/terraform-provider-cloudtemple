package client

import "context"

type NetworkClient struct {
	c *Client
}

func (c *ComputeClient) Network() *NetworkClient {
	return &NetworkClient{c.c}
}

type NetworkFilter struct {
	Name             string `filter:"name"`
	MachineManagerId string `filter:"machineManagerId"`
	DatacenterId     string `filter:"datacenterId"`
	VirtualMachineId string `filter:"virtualMachineId"`
	Type             string `filter:"type"`
	VirtualSwitchId  string `filter:"virtualSwitchId"`
	HostId           string `filter:"hostId"`
	HostClusterId    string `filter:"hostClusterId"`
	FolderId         string `filter:"folderId"`
}

type Network struct {
	ID                    string   `terraform:"id"`
	Name                  string   `terraform:"name"`
	Moref                 string   `terraform:"moref"`
	MachineManagerId      string   `terraform:"machine_manager_id"`
	VirtualMachinesNumber int      `terraform:"virtual_machines_number"`
	HostNumber            int      `terraform:"host_number"`
	HostNames             []string `terraform:"host_names"`
}

func (n *NetworkClient) List(
	ctx context.Context,
	filter *NetworkFilter) ([]*Network, error) {

	r := n.c.newRequest("GET", "/compute/v1/vcenters/networks")
	r.addFilter(filter)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Network
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (n *NetworkClient) Read(ctx context.Context, id string) (*Network, error) {
	r := n.c.newRequest("GET", "/compute/v1/vcenters/networks/%s", id)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out Network
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
