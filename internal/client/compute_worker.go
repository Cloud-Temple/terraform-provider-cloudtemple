package client

import "context"

type WorkerClient struct {
	c *Client
}

func (c *Compute) Worker() *WorkerClient {
	return &WorkerClient{c.c}
}

type Worker struct {
	ID                    string
	Name                  string
	FullName              string
	Vendor                string
	Version               string
	Build                 int
	LocaleVersion         string
	LocaleBuild           int
	OsType                string
	ProductLineID         string
	ApiType               string
	ApiVersion            string
	InstanceUuid          string
	LicenseProductName    string
	LicenseProductVersion int
	TenantID              string
	TenantName            string
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
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out Worker
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
