package client

import "context"

// VPCPrivateNetworkClient handles read operations on VPC private networks.
type VPCPrivateNetworkClient struct {
	c *Client
}

// PrivateNetwork represents a private network as returned by
// GET /vpc/v1/private_networks[/{id}].
//
// ipAddress carries the network address in CIDR notation (e.g. 192.168.1.0/24).
// name is nullable in the API. The gateway is NOT exposed by the API yet and is
// therefore intentionally absent.
type PrivateNetwork struct {
	ID            string     `json:"id"`
	IPAddress     string     `json:"ipAddress"`
	Name          *string    `json:"name"`
	VlanID        int        `json:"vlanId"`
	StaticIPCount int        `json:"staticIpCount"`
	VPC           BaseObject `json:"vpc"`
}

// PrivateNetworkFilter narrows a private-network listing.
type PrivateNetworkFilter struct {
	VpcID string `filter:"vpcId"`
}

// List retrieves private networks, optionally filtered by VPC ID.
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

// Read retrieves a single private network by ID. It returns (nil, nil) when the
// private network does not exist (404).
func (p *VPCPrivateNetworkClient) Read(ctx context.Context, id string) (*PrivateNetwork, error) {
	r := p.c.newRequest("GET", "/vpc/v1/private_networks/%s", id)
	resp, err := p.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PrivateNetwork
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
