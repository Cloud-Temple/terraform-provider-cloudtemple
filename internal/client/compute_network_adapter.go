package client

import "context"

type NetworkAdapterClient struct {
	c *Client
}

func (c *ComputeClient) NetworkAdapter() *NetworkAdapterClient {
	return &NetworkAdapterClient{c.c}
}

type NetworkAdapter struct {
	ID               string `terraform:"id"`
	VirtualMachineId string `terraform:"virtual_machine_id"`
	Name             string `terraform:"name"`
	NetworkId        string `terraform:"network_id"`
	Type             string `terraform:"type"`
	MacType          string `terraform:"mac_type"`
	MacAddress       string `terraform:"mac_address"`
	Connected        bool   `terraform:"connected"`
	AutoConnect      bool   `terraform:"auto_connect"`
}

func (n *NetworkAdapterClient) List(ctx context.Context, virtualMachineId string) ([]*NetworkAdapter, error) {
	r := n.c.newRequest("GET", "/compute/v1/vcenters/network_adapters")
	r.params.Add("virtualMachineId", virtualMachineId)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*NetworkAdapter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type CreateNetworkAdapterRequest struct {
	VirtualMachineId string `json:"virtualMachineId"`
	NetworkId        string `json:"networkId"`
	Type             string `json:"type"`
	MacAddress       string `json:"macAddress,omitempty"`
}

func (n *NetworkAdapterClient) Create(ctx context.Context, req *CreateNetworkAdapterRequest) (string, error) {
	r := n.c.newRequest("POST", "/compute/v1/vcenters/network_adapters")
	r.obj = req
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *NetworkAdapterClient) Read(ctx context.Context, id string) (*NetworkAdapter, error) {
	r := n.c.newRequest("GET", "/compute/v1/vcenters/network_adapters/%s", id)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out NetworkAdapter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type UpdateNetworkAdapterRequest struct {
	ID           string `json:"id"`
	NewNetworkId string `json:"newNetworkId"`
	AutoConnect  bool   `json:"autoConnect"`
	MacAddress   string `json:"macAddress,omitempty"`
}

func (n *NetworkAdapterClient) Update(ctx context.Context, req *UpdateNetworkAdapterRequest) (string, error) {
	r := n.c.newRequest("PATCH", "/compute/v1/vcenters/network_adapters")
	r.obj = req
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *NetworkAdapterClient) Delete(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("DELETE", "/compute/v1/vcenters/network_adapters/%s", id)
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *NetworkAdapterClient) Connect(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("PATCH", "/compute/v1/vcenters/network_adapters/connect")
	r.obj = map[string]string{"id": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *NetworkAdapterClient) Disconnect(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("PATCH", "/compute/v1/vcenters/network_adapters/disconnect")
	r.obj = map[string]string{"id": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}
