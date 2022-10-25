package client

import "context"

type WorkerClient struct {
	c *Client
}

func (c *Compute) Worker() *WorkerClient {
	return &WorkerClient{c.c}
}

type Worker struct {
	ID                    string `terraform:"id"`
	Name                  string `terraform:"name"`
	FullName              string `terraform:"full_name"`
	Vendor                string `terraform:"vendor"`
	Version               string `terraform:"version"`
	Build                 int    `terraform:"build"`
	LocaleVersion         string `terraform:"locale_version"`
	LocaleBuild           int    `terraform:"locale_build"`
	OsType                string `terraform:"os_type"`
	ProductLineID         string `terraform:"product_line_id"`
	ApiType               string `terraform:"api_type"`
	ApiVersion            string `terraform:"api_version"`
	InstanceUuid          string `terraform:"instance_uuid"`
	LicenseProductName    string `terraform:"license_product_name"`
	LicenseProductVersion int    `terraform:"license_product_version"`
	TenantID              string `terraform:"tenant_id"`
	TenantName            string `terraform:"tenant_name"`
}

func (v *WorkerClient) List(ctx context.Context, name string) ([]*Worker, error) {
	// TODO: filters
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters")
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Worker
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (v *WorkerClient) Read(ctx context.Context, id string) (*Worker, error) {
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/"+id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out Worker
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
