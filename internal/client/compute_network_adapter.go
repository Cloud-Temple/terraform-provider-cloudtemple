package client

import "context"

type NetworkAdapterClient struct {
	c *Client
}

func (c *Compute) NetworkAdapter() *NetworkAdapterClient {
	return &NetworkAdapterClient{c.c}
}

type NetworkAdapter struct {
	ID               string
	VirtualMachineId string
	Name             string
	Type             string
	MacType          string
	MacAddress       string
	Connected        bool
	AutoConnect      bool
}

func (n *NetworkAdapterClient) List(ctx context.Context, virtualMachineId string) ([]*NetworkAdapter, error) {
	r := n.c.newRequest("GET", "/api/compute/v1/vcenters/network_adapters")
	r.params.Add("virtualMachineId", virtualMachineId)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*NetworkAdapter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (n *NetworkAdapterClient) Read(ctx context.Context, id string) (*NetworkAdapter, error) {
	r := n.c.newRequest("GET", "/api/compute/v1/vcenters/network_adapters/"+id)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out NetworkAdapter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
