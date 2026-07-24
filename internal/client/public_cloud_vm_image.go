package client

import "context"

type PublicCloudVMImageClient struct {
	c *Client
}

// Image returns the image (OS image) catalogue sub-client (read-only).
func (v *PublicCloudVMClient) Image() *PublicCloudVMImageClient {
	return &PublicCloudVMImageClient{v.c}
}

// PublicCloudVMImage mirrors an element of GET /vm_instances/v1/images
// (bare array, camelCase). Only the stable/core fields are modelled here; the
// editorial marketplace fields (details_en, deployment_options, ...) and the
// networkInterfaces block are intentionally left out of the provider surface
// for iteration 1 (nested shapes to confirm live). The struct carries no json
// tags: encoding/json matches the camelCase keys case-insensitively, so the Go
// field ImageType matches the API's imageType without an explicit tag.
type PublicCloudVMImage struct {
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
	ImageType          string
	Icon               string
}

// PublicCloudVMImageFilter carries the optional list filters.
type PublicCloudVMImageFilter struct {
	InstanceFamilyID   string `filter:"instanceFamilyId"`
	AvailabilityZoneID string `filter:"availabilityZoneId"`
}

// List returns the images of the tenant (bare JSON array), optionally filtered
// by instance family and/or availability zone.
func (t *PublicCloudVMImageClient) List(ctx context.Context, filter *PublicCloudVMImageFilter) ([]*PublicCloudVMImage, error) {
	req := t.c.newRequest("GET", "/vm_instances/v1/images")
	req.addFilter(filter)
	resp, err := t.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*PublicCloudVMImage
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read returns a single image by UUID. A positive 404 maps to (nil, nil); any
// other non-OK code (403, 5xx) fails closed with an error.
func (t *PublicCloudVMImageClient) Read(ctx context.Context, id string) (*PublicCloudVMImage, error) {
	req := t.c.newRequest("GET", "/vm_instances/v1/images/%s", id)
	resp, err := t.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMImage
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
