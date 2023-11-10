package client

import "context"

type FeatureClient struct {
	c *Client
}

func (i *IAM) Feature() *FeatureClient {
	return &FeatureClient{i.c}
}

type Feature struct {
	ID   string `terraform:"id"`
	Name string `terraform:"name"`

	SubFeatures []*Feature `terraform:"subfeatures"`
}

func (f *FeatureClient) List(ctx context.Context) ([]*Feature, error) {
	r := f.c.newRequest("GET", "/iam/v2/features")
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Feature
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type FeatureAssignment struct {
	FeatureID string
	TenantID  string
}

func (f *FeatureClient) ListAssignments(ctx context.Context, tenantID string) ([]*FeatureAssignment, error) {
	r := f.c.newRequest("GET", "/iam/v2/features/assignments")
	r.params.Set("tenantId", tenantID)
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*FeatureAssignment
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
