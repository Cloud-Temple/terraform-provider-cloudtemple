package client

import "context"

type RoleClient struct {
	c *Client
}

func (i *IAM) Role() *RoleClient {
	return &RoleClient{i.c}
}

type Role struct {
	ID   string
	Name string
}

func (r *RoleClient) List(ctx context.Context) ([]*Role, error) {
	req := r.c.newRequest("GET", "/api/iam/v2/roles")
	resp, err := r.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Role
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *RoleClient) Read(ctx context.Context, roleID string) (*Role, error) {
	req := r.c.newRequest("GET", "/api/iam/v2/roles/"+roleID)
	resp, err := r.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out Role
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
