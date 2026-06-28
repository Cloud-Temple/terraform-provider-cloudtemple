package client

import "context"

type OpenIaaSNetworkClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) Network() *OpenIaaSNetworkClient {
	return &OpenIaaSNetworkClient{c.c.c}
}

type OpenIaaSNetwork struct {
	ID                         string
	MachineManager             BaseObject
	InternalID                 string
	Name                       string
	Pool                       BaseObject
	MaximumTransmissionUnit    int
	NetworkAdapters            []string
	NetworkBlockDevice         bool
	InsecureNetworkBlockDevice bool
	// VPC is the network's VPC association (swagger.comput.yml openIaasNetwork
	// `vpc`). It is a POINTER: the API emits the `vpc` object only for a
	// VPC-backed network, so a plain network decodes it to nil. The provider
	// uses VPC != nil to tell a VPC-backed network apart from a plain one
	// before assigning a static IP (#1854).
	VPC *OpenIaaSNetworkAdapterVPC `json:"vpc"`
}

type OpenIaaSNetworkFilter struct {
	MachineManagerID string `filter:"machineManagerId"`
	PoolID           string `filter:"poolId"`
}

func (n *OpenIaaSNetworkClient) List(
	ctx context.Context,
	filter *OpenIaaSNetworkFilter) ([]*OpenIaaSNetwork, error) {

	r := n.c.newRequest("GET", "/compute/v1/open_iaas/networks")
	r.addFilter(filter)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*OpenIaaSNetwork
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (n *OpenIaaSNetworkClient) Read(ctx context.Context, id string) (*OpenIaaSNetwork, error) {
	r := n.c.newRequest("GET", "/compute/v1/open_iaas/networks/%s", id)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSNetwork
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
