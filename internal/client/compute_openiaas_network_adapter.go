package client

import "context"

type OpenIaaSNetworkAdapterClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) NetworkAdapter() *OpenIaaSNetworkAdapterClient {
	return &OpenIaaSNetworkAdapterClient{c.c.c}
}

type OpenIaaSNetworkAdapter struct {
	ID               string `terraform:"id"`
	Name             string `terraform:"name"`
	MachineManagerID string `terraform:"machine_manager_id"`
	InternalID       string `terraform:"internal_id"`
	VirtualMachineID string `terraform:"virtual_machine_id"`
	MacAddress       string `terraform:"mac_address"`
	MTU              int    `terraform:"mtu"`
	Attached         bool   `terraform:"attached"`
	Network          struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
	} `terraform:"network"`
}

func (v *OpenIaaSNetworkAdapterClient) Read(ctx context.Context, id string) (*OpenIaaSNetworkAdapter, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/network_adapters/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSNetworkAdapter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (v *OpenIaaSNetworkAdapterClient) List(ctx context.Context, virtualMachineId string) ([]*OpenIaaSNetworkAdapter, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/network_adapters")
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

	var out []*OpenIaaSNetworkAdapter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
