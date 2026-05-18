package client

import "context"

// VPCFloatingIPClient handles floating IP operations
type VPCFloatingIPClient struct {
	c *Client
}

// FloatingIP represents a floating IP
type FloatingIP struct {
	ID          string `json:"id"`
	IPAddress   string `json:"ipAddress"`
	Description string `json:"description"`
	StaticIP    *struct {
		ID      string `json:"id"`
		Address string `json:"address"`
	} `json:"staticIp"`
	VPC            *BaseObject `json:"vpc"`
	PrivateNetwork *BaseObject `json:"privateNetwork"`
}

// FloatingIPFilter represents the filter for listing floating IPs
type FloatingIPFilter struct {
	VpcID string `filter:"vpcId"`
}

// List retrieves all floating IPs, optionally filtered by VPC ID
func (f *VPCFloatingIPClient) List(ctx context.Context, filter *FloatingIPFilter) ([]*FloatingIP, error) {
	r := f.c.newRequest("GET", "/vpc/v1/floating_ips")
	r.addFilter(filter)
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*FloatingIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read retrieves a specific floating IP by ID
func (f *VPCFloatingIPClient) Read(ctx context.Context, id string) (*FloatingIP, error) {
	r := f.c.newRequest("GET", "/vpc/v1/floating_ips/%s", id)
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out FloatingIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
