package client

import "context"

type PublicCloudVMTemplateClient struct {
	c *Client
}

// Template returns the template (OS image) catalogue sub-client (read-only).
func (v *PublicCloudVMClient) Template() *PublicCloudVMTemplateClient {
	return &PublicCloudVMTemplateClient{v.c}
}

// PublicCloudVMTemplate mirrors an element of GET /vm_instances/v1/templates
// (bare array, camelCase). Only the stable/core fields are modelled here; the
// editorial marketplace fields (details_en, deployment_options, ...) and the
// networkInterfaces block are intentionally left out of the provider surface
// for iteration 1 (nested shapes to confirm live).
type PublicCloudVMTemplate struct {
	ID                 string
	Name               string
	OsFamily           string
	OsName             string
	OsVersion          string
	DiskSizesGb        []int
	CompatibleFamilies []string
	Categories         []string
	Family             string
	Version            string
	Editor             string
	DescriptionEn      string
	TemplateType       string
	Icon               string
}

// PublicCloudVMTemplateFilter carries the optional list filters.
type PublicCloudVMTemplateFilter struct {
	InstanceFamilyID   string `filter:"instanceFamilyId"`
	AvailabilityZoneID string `filter:"availabilityZoneId"`
}

// List returns the templates of the tenant (bare JSON array), optionally filtered
// by instance family and/or availability zone.
func (t *PublicCloudVMTemplateClient) List(ctx context.Context, filter *PublicCloudVMTemplateFilter) ([]*PublicCloudVMTemplate, error) {
	req := t.c.newRequest("GET", "/vm_instances/v1/templates")
	req.addFilter(filter)
	resp, err := t.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*PublicCloudVMTemplate
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read returns a single template by UUID. A positive 404 maps to (nil, nil); any
// other non-OK code (403, 5xx) fails closed with an error.
func (t *PublicCloudVMTemplateClient) Read(ctx context.Context, id string) (*PublicCloudVMTemplate, error) {
	req := t.c.newRequest("GET", "/vm_instances/v1/templates/%s", id)
	resp, err := t.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMTemplate
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
