package client

import "context"

type PublicCloudVMFlavorClient struct {
	c *Client
}

// Flavor returns the flavor catalogue sub-client (read-only, list-only: the API
// has no by-id endpoint for flavors).
func (v *PublicCloudVMClient) Flavor() *PublicCloudVMFlavorClient {
	return &PublicCloudVMFlavorClient{v.c}
}

// PublicCloudVMFlavor is a predefined (vCPU, RAM) sizing pair of an instance
// family. It is NOT consumed directly by the VM resource (which sizes by
// instance_family_id + cpu + memory); it exists to discover valid combinations.
type PublicCloudVMFlavor struct {
	ID               string
	InstanceFamilyID string
	Name             string
	Vcpu             int
	RamGb            int
}

// PublicCloudVMFlavorFilter carries the optional list filters.
type PublicCloudVMFlavorFilter struct {
	FamilyID string `filter:"familyId"`
}

// publicCloudVMFlavorListResponse mirrors the wrapped shape of
// GET /vm_instances/v1/flavors, which returns { "flavors": [ ... ] } (unlike the
// bare-array region/availability_zone endpoints).
type publicCloudVMFlavorListResponse struct {
	Flavors []*PublicCloudVMFlavor
}

// List returns the flavors of the tenant, optionally filtered by instance family.
func (f *PublicCloudVMFlavorClient) List(ctx context.Context, filter *PublicCloudVMFlavorFilter) ([]*PublicCloudVMFlavor, error) {
	req := f.c.newRequest("GET", "/vm_instances/v1/flavors")
	req.addFilter(filter)
	resp, err := f.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out publicCloudVMFlavorListResponse
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out.Flavors, nil
}
