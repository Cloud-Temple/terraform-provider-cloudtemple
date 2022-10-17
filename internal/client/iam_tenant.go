package client

import "context"

type TenantClient struct {
	c *Client
}

func (i *IAM) Tenant() *TenantClient {
	return &TenantClient{i.c}
}

type Tenant struct {
	ID        string
	Name      string
	SNC       bool
	CompanyID string
}

func (t *TenantClient) List(ctx context.Context, companyID string) ([]*Tenant, error) {
	r := t.c.newRequest("GET", "/api/iam/v2/tenants")
	resp, err := t.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Tenant
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
