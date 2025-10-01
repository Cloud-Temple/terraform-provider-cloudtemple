package client

import "context"

type OpenIaaSNetworkAdapterClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) NetworkAdapter() *OpenIaaSNetworkAdapterClient {
	return &OpenIaaSNetworkAdapterClient{c.c.c}
}

type OpenIaaSNetworkAdapter struct {
	ID               string
	Name             string
	InternalID       string
	VirtualMachineID string
	MacAddress       string
	MTU              int
	Attached         bool
	TxChecksumming   bool
	Network          BaseObject
	MachineManager   BaseObject
}

type CreateOpenIaasNetworkAdapterRequest struct {
	VirtualMachineID string `json:"virtualMachineId"`
	NetworkID        string `json:"networkId"`
	MAC              string `json:"mac,omitempty"`
}

func (v *OpenIaaSNetworkAdapterClient) Create(ctx context.Context, req *CreateOpenIaasNetworkAdapterRequest) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/open_iaas/network_adapters")
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
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

type OpenIaaSNetworkAdapterFilter struct {
	VirtualMachineID string `filter:"virtualMachineId"`
}

func (v *OpenIaaSNetworkAdapterClient) List(ctx context.Context, filter *OpenIaaSNetworkAdapterFilter) ([]*OpenIaaSNetworkAdapter, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/network_adapters")
	r.addFilter(filter)
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

type UpdateOpenIaasNetworkAdapterRequest struct {
	NetworkID      string `json:"networkId"`
	MAC            string `json:"mac"`
	Attached       bool   `json:"attached"`
	TxChecksumming bool   `json:"txChecksumming"`
}

func (v *OpenIaaSNetworkAdapterClient) Update(ctx context.Context, id string, req *UpdateOpenIaasNetworkAdapterRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/network_adapters/%s", id)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSNetworkAdapterClient) Connect(ctx context.Context, id string) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/network_adapters/%s/connect", id)
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSNetworkAdapterClient) Disconnect(ctx context.Context, id string) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/network_adapters/%s/disconnect", id)
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSNetworkAdapterClient) Delete(ctx context.Context, id string) (string, error) {
	r := v.c.newRequest("DELETE", "/compute/v1/open_iaas/network_adapters/%s", id)
	return v.c.doRequestAndReturnActivity(ctx, r)
}
