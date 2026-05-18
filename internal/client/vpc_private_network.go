package client

import "context"

// VPCPrivateNetworkClient handles private network operations
type VPCPrivateNetworkClient struct {
	c *Client
}

// PrivateNetwork represents a private network in a VPC
type PrivateNetwork struct {
	ID            string     `json:"id"`
	IPAddress     string     `json:"ipAddress"`
	Name          *string    `json:"name"`
	VlanID        int        `json:"vlanId"`
	StaticIPCount int        `json:"staticIpCount"`
	VPC           BaseObject `json:"vpc"`
}

// PrivateNetworkFilter represents the filter for listing private networks
type PrivateNetworkFilter struct {
	VpcID string `filter:"vpcId"`
}

// List retrieves all private networks, optionally filtered by VPC ID
func (p *VPCPrivateNetworkClient) List(ctx context.Context, filter *PrivateNetworkFilter) ([]*PrivateNetwork, error) {
	r := p.c.newRequest("GET", "/vpc/v1/private_networks")
	r.addFilter(filter)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*PrivateNetwork
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read retrieves a specific private network by ID
func (p *VPCPrivateNetworkClient) Read(ctx context.Context, id string) (*PrivateNetwork, error) {
	r := p.c.newRequest("GET", "/vpc/v1/private_networks/%s", id)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out PrivateNetwork
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
