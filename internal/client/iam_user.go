package client

import "context"

type UserClient struct {
	c *Client
}

func (i *IAM) User() *UserClient {
	return &UserClient{i.c}
}

type User struct {
	ID            string   `terraform:"id"`
	InternalID    string   `terraform:"internal_id"`
	Name          string   `terraform:"name"`
	Type          string   `terraform:"type"`
	Source        []string `terraform:"source"`
	SourceID      string   `terraform:"source_id"`
	EmailVerified bool     `terraform:"email_verified"`
	Email         string   `terraform:"email"`
}

func (t *UserClient) Read(ctx context.Context, userID string) (*User, error) {
	r := t.c.newRequest("GET", "/iam/v2/users/%s", userID)
	resp, err := t.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out User
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type UserFilter struct {
	CompanyID string `filter:"companyId"`
}

func (t *UserClient) List(ctx context.Context, filter *UserFilter) ([]*User, error) {
	r := t.c.newRequest("GET", "/iam/v2/users")
	r.addFilter(filter)
	resp, err := t.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*User
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
