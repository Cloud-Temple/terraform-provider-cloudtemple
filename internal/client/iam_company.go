package client

import "context"

type CompanyClient struct {
	c *Client
}

func (i *IAM) Company() *CompanyClient {
	return &CompanyClient{i.c}
}

type Company struct {
	ID   string
	Name string
}

func (c *CompanyClient) Read(ctx context.Context, companyID string) (*Company, error) {
	r := c.c.newRequest("GET", "/api/iam/v2/companies/"+companyID)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out Company
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
