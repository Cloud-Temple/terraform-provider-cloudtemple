package client

import (
	"context"
)

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

// CreateFloatingIPRequest represents a request to create a floating IP
type CreateFloatingIPRequest struct {
	Count int `json:"count"`
}

// Create creates a new floating IP
// Returns the activity ID that should be waited for completion
func (f *VPCFloatingIPClient) Create(ctx context.Context, req *CreateFloatingIPRequest) (string, error) {
	r := f.c.newRequest("POST", "/vpc/v1/floating_ips")
	r.obj = req
	return f.c.doRequestAndReturnActivity(ctx, r)
}

// UpdateFloatingIPRequest represents a request to update a floating IP
type UpdateFloatingIPRequest struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Update updates a floating IP
// Returns the activity ID that should be waited for completion
func (f *VPCFloatingIPClient) Update(ctx context.Context, req *UpdateFloatingIPRequest) (string, error) {
	r := f.c.newRequest("PATCH", "/vpc/v1/floating_ips/%s", req.ID)
	r.obj = req
	return f.c.doRequestAndReturnActivity(ctx, r)
}

// Bind binds a floating IP to a static IP
// Returns the activity ID that should be waited for completion
func (f *VPCFloatingIPClient) Bind(ctx context.Context, floatingIPID, staticIPID string) (string, error) {
	r := f.c.newRequest("POST", "/vpc/v1/floating_ips/%s/bind/static_ips/%s", floatingIPID, staticIPID)
	return f.c.doRequestAndReturnActivity(ctx, r)
}

// Unbind unbinds a floating IP from a static IP
// Returns the activity ID that should be waited for completion
func (f *VPCFloatingIPClient) Unbind(ctx context.Context, floatingIPID, staticIPID string) (string, error) {
	r := f.c.newRequest("DELETE", "/vpc/v1/floating_ips/%s/unbind/static_ips/%s", floatingIPID, staticIPID)
	return f.c.doRequestAndReturnActivity(ctx, r)
}

// Delete deletes a floating IP
// Returns the activity ID that should be waited for completion
func (f *VPCFloatingIPClient) Delete(ctx context.Context, id string) (string, error) {
	r := f.c.newRequest("DELETE", "/vpc/v1/floating_ips/%s", id)
	return f.c.doRequestAndReturnActivity(ctx, r)
}
