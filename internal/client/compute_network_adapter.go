package client

import "context"

type NetworkAdapterClient struct {
	c *Client
}

func (c *ComputeClient) NetworkAdapter() *NetworkAdapterClient {
	return &NetworkAdapterClient{c.c}
}

type NetworkAdapter struct {
	ID               string
	VirtualMachineId string
	Name             string
	Network          BaseObject
	Type             string
	MacType          string
	MacAddress       string
	Connected        bool
	AutoConnect      bool
	// VPC carries the VPC association of the adapter (vpcDetails, shared with the
	// OpenIaaS shape). POINTER: the API emits the `vpc` object only when the
	// adapter is on a VPC-backed network; a plain-network adapter decodes it to
	// nil (#375, confirmed live on the recette vCenter).
	VPC *OpenIaaSNetworkAdapterVPC `json:"vpc"`
}

type NetworkAdapterFilter struct {
	VirtualMachineID string `filter:"virtualMachineId"`
}

func (n *NetworkAdapterClient) List(ctx context.Context, filter *NetworkAdapterFilter) ([]*NetworkAdapter, error) {
	r := n.c.newRequest("GET", "/compute/v1/vcenters/network_adapters")
	r.addFilter(filter)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out []*NetworkAdapter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// ListStrict behaves like List but requires a complete HTTP 200 answer: 206 is
// a partial listing and cannot prove an absence, and any other code (including
// 403) is an error rather than being mapped to an empty result. Callers using
// the listing as state-safety evidence must fail closed on anything else
// (#281).
func (n *NetworkAdapterClient) ListStrict(ctx context.Context, filter *NetworkAdapterFilter) ([]*NetworkAdapter, error) {
	r := n.c.newRequest("GET", "/compute/v1/vcenters/network_adapters")
	r.addFilter(filter)
	resp, err := n.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireHttpCodes(resp, 200); err != nil {
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
	// IPAddress is the VPC static IP to assign when the target network is
	// VPC-backed (compute API #1854, source=vmware). Ignored on a plain network.
	IPAddress string `json:"ipAddress,omitempty"`
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
	found, err := requireNotFoundOrOK(resp, 404)
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
	// IPAddress relocates the VPC static IP of a VPC-backed adapter to this
	// address (compute API #1854, source=vmware). Sent only by the VPC IP
	// reconciliation step, on a genuine divergence.
	IPAddress string `json:"ipAddress,omitempty"`
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
