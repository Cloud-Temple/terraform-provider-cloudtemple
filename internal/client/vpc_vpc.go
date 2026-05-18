package client

import "context"

// VPCVPCClient handles VPC operations
type VPCVPCClient struct {
	c *Client
}

// VPC represents a VPC
type VPC struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	InternetIP          string `json:"internetIp"`
	PrivateNetworkCount int    `json:"privateNetworkCount"`
	StaticIPCount       int    `json:"staticIpCount"`
	FloatingIPCount     int    `json:"floatingIpCount"`
}

// List retrieves all VPCs
func (v *VPCVPCClient) List(ctx context.Context) ([]*VPC, error) {
	r := v.c.newRequest("GET", "/vpc/v1/vpc")
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*VPC
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read retrieves a specific VPC by ID
func (v *VPCVPCClient) Read(ctx context.Context, id string) (*VPC, error) {
	r := v.c.newRequest("GET", "/vpc/v1/vpc/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out VPC
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
