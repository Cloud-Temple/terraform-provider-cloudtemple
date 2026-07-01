package client

import "context"

type PublicCloudVMRegionClient struct {
	c *Client
}

// Region returns the region catalogue sub-client (read-only).
func (v *PublicCloudVMClient) Region() *PublicCloudVMRegionClient {
	return &PublicCloudVMRegionClient{v.c}
}

// PublicCloudVMRegion mirrors an element of GET /vm_instances/v1/regions. The
// broker camelizes responses (countryCode, azCount, isEnabled, ...); Go's
// encoding/json matches struct fields case-insensitively, so no json tags are
// needed. Description is nullable in the API and decodes to "" when null.
type PublicCloudVMRegion struct {
	ID          string
	Name        string
	Description string
	CountryCode string
	Geography   string
	IsEnabled   bool
	AzCount     int
	CreatedAt   string
	UpdatedAt   string
}

// List returns every region of the tenant (bare JSON array, no server-side
// filter or pagination on this endpoint).
func (r *PublicCloudVMRegionClient) List(ctx context.Context) ([]*PublicCloudVMRegion, error) {
	req := r.c.newRequest("GET", "/vm_instances/v1/regions")
	resp, err := r.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*PublicCloudVMRegion
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read returns a single region by UUID. A positive 404 maps to (nil, nil)
// (absence); any other non-OK code (403, 5xx) is returned as an error so the
// caller fails closed (never silently treats a permission/backend blip as
// absence).
func (r *PublicCloudVMRegionClient) Read(ctx context.Context, id string) (*PublicCloudVMRegion, error) {
	req := r.c.newRequest("GET", "/vm_instances/v1/regions/%s", id)
	resp, err := r.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMRegion
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
