package client

import "context"

// VPCFloatingIPClient handles read operations on VPC floating IPs.
type VPCFloatingIPClient struct {
	c *Client
}

// FloatingIPStaticIP is the static IP a floating IP is bound to, as nested in
// the FloatingIp schema. The API names the address field "address" here
// (distinct from StaticIP.FloatingIP which uses "ipAddress").
type FloatingIPStaticIP struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// FloatingIP represents a floating IP as returned by
// GET /vpc/v1/floating_ips[/{id}].
//
// staticIp, vpc and privateNetwork are nullable in the API: they are populated
// only when the floating IP is bound to a static IP.
type FloatingIP struct {
	ID             string              `json:"id"`
	IPAddress      string              `json:"ipAddress"`
	Description    string              `json:"description"`
	StaticIP       *FloatingIPStaticIP `json:"staticIp"`
	VPC            *BaseObject         `json:"vpc"`
	PrivateNetwork *BaseObject         `json:"privateNetwork"`
}

// FloatingIPFilter narrows a floating-IP listing.
type FloatingIPFilter struct {
	VpcID string `filter:"vpcId"`
}

// List retrieves floating IPs, optionally filtered by VPC ID.
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

// Read retrieves a single floating IP by ID. It returns (nil, nil) when the
// floating IP does not exist (403; the API returns 403 for an absent resource).
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
