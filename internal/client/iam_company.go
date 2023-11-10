package client

import (
	"context"
)

type CompanyClient struct {
	c *Client
}

func (i *IAM) Company() *CompanyClient {
	return &CompanyClient{i.c}
}

type Company struct {
	ID   string `terraform:"id"`
	Name string `terraform:"name"`
}

func (c *CompanyClient) Read(ctx context.Context, companyID string) (*Company, error) {
	r := c.c.newRequest("GET", "/iam/v2/companies/%s", companyID)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out Company
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
