package client

import "context"

type PublicCloudVMAvailabilityZoneClient struct {
	c *Client
}

// AvailabilityZone returns the availability-zone catalogue sub-client (read-only).
func (v *PublicCloudVMClient) AvailabilityZone() *PublicCloudVMAvailabilityZoneClient {
	return &PublicCloudVMAvailabilityZoneClient{v.c}
}

// PublicCloudVMFamilyRef is a summary reference to an instance family ({id, name}),
// as embedded in an availability zone's compatibleFamilies.
type PublicCloudVMFamilyRef struct {
	ID   string
	Name string
}

// PublicCloudVMAvailabilityZone mirrors an element of
// GET /vm_instances/v1/availability_zones. camelCase is matched
// case-insensitively by encoding/json; description is nullable (decodes to "").
type PublicCloudVMAvailabilityZone struct {
	ID                 string
	Name               string
	Description        string
	RegionID           string
	IsEnabled          bool
	CompatibleFamilies []PublicCloudVMFamilyRef
	CreatedAt          string
	UpdatedAt          string
}

// PublicCloudVMAvailabilityZoneFilter carries the optional list filters.
type PublicCloudVMAvailabilityZoneFilter struct {
	RegionID string `filter:"regionId"`
}

// List returns the availability zones of the tenant (bare JSON array), optionally
// filtered by region.
func (a *PublicCloudVMAvailabilityZoneClient) List(ctx context.Context, filter *PublicCloudVMAvailabilityZoneFilter) ([]*PublicCloudVMAvailabilityZone, error) {
	req := a.c.newRequest("GET", "/vm_instances/v1/availability_zones")
	req.addFilter(filter)
	resp, err := a.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*PublicCloudVMAvailabilityZone
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read returns a single availability zone by UUID. A positive 404 maps to
// (nil, nil); any other non-OK code (403, 5xx) fails closed with an error.
func (a *PublicCloudVMAvailabilityZoneClient) Read(ctx context.Context, id string) (*PublicCloudVMAvailabilityZone, error) {
	req := a.c.newRequest("GET", "/vm_instances/v1/availability_zones/%s", id)
	resp, err := a.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMAvailabilityZone
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
