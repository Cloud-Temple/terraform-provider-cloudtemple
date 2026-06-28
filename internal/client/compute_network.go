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
	ID                    string
	Name                  string
	Moref                 string
	MachineManager        BaseObject
	VirtualMachinesNumber int
	HostNumber            int
	HostNames             []string
	// VPC is the network's VPC association. POINTER: the API emits the `vpc`
	// object only for a VPC-backed (vStack/VPC) portgroup, so a plain network
	// decodes it to nil. Used to reject ip_address on a non-VPC network before
	// any side effect (#375, confirmed live: the listing exposes `vpc`).
	VPC *OpenIaaSNetworkAdapterVPC `json:"vpc"`
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
