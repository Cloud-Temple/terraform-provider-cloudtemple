package client

import "context"

// VPCVPCClient handles read operations on VPCs.
type VPCVPCClient struct {
	c *Client
}

// VPC represents a VPC as returned by GET /vpc/v1/vpc[/{id}].
//
// Field names and JSON tags follow the Shiva VPC API swagger (v0.15.0).
// internetIp is nullable in the API, so it is modelled as *string to
// distinguish "no internet IP" from an empty value.
type VPC struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	InternetIP          *string `json:"internetIp"`
	PrivateNetworkCount int     `json:"privateNetworkCount"`
	StaticIPCount       int     `json:"staticIpCount"`
	FloatingIPCount     int     `json:"floatingIpCount"`
}

// List retrieves all VPCs the caller can read.
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

// Read retrieves a single VPC by ID. It returns (nil, nil) when the VPC does
// not exist (403; the API returns 403 for an absent resource) so callers can surface a precise not-found error.
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
