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
	// IPv4Address / IPv6Address are the adapter's IP addresses as reported by
	// the platform (swagger.comput.yml openIaasNetworkAdapter). They are
	// READ-ONLY: the provider never writes them.
	IPv4Address string `json:"ipv4Address"`
	IPv6Address string `json:"ipv6Address"`
	// VPC carries the VPC association of the adapter. It is a POINTER because
	// the API only emits the `vpc` object when the adapter is on a VPC
	// network; an adapter on a plain network decodes it to nil (#238).
	VPC *OpenIaaSNetworkAdapterVPC `json:"vpc"`
}

// OpenIaaSNetworkAdapterVPC mirrors the API `vpcDetails` schema attached to an
// OpenIaaS network adapter (swagger.comput.yml). Only the NON-deprecated
// nested privateNetwork{id,name} is decoded; the deprecated top-level
// privateNetworkId / privateNetworkName are intentionally ignored (#238).
type OpenIaaSNetworkAdapterVPC struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	PrivateNetwork struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"privateNetwork"`
	StaticIPAddress string `json:"staticIpAddress"`
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

// ListStrict behaves like List but treats an access-denied answer as an
// error instead of an empty result: callers using the listing as EVIDENCE
// for state-shrinking decisions must fail closed (#273).
func (v *OpenIaaSNetworkAdapterClient) ListStrict(ctx context.Context, filter *OpenIaaSNetworkAdapterFilter) ([]*OpenIaaSNetworkAdapter, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/network_adapters")
	r.addFilter(filter)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	// Strictly 200: a 206 partial listing cannot prove an absence.
	if err := requireHttpCodes(resp, 200); err != nil {
		return nil, err
	}

	var out []*OpenIaaSNetworkAdapter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
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
	// All fields are optional (PATCH semantics): callers only set the
	// fields that actually diverge — re-sending the current networkId/mac
	// is rejected platform-side as a VPC Static IP self-conflict (#246).
	NetworkID string `json:"networkId,omitempty"`
	MAC       string `json:"mac,omitempty"`
	Attached  bool   `json:"attached,omitempty"`
	// Pointer so that an explicit `false` is serialized: with a plain bool
	// and omitempty, disabling TX checksumming could never be sent (#246).
	TxChecksumming *bool `json:"txChecksumming,omitempty"`
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
