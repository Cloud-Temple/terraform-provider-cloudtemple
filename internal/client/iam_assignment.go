package client

import "context"

type AssignmentClient struct {
	c *Client
}

func (i *IAM) Assignment() *AssignmentClient {
	return &AssignmentClient{i.c}
}

type TenantAssignment struct {
	UserID   string
	TenantID string
	RoleID   string
}

func (a *AssignmentClient) List(ctx context.Context, userId, tenantId, roleId string) ([]*TenantAssignment, error) {
	r := a.c.newRequest("GET", "/api/iam/v2/assignments/tenant")
	r.params.Set("userId", userId)
	r.params.Set("tenantId", tenantId)
	if roleId != "" {
		r.params.Set("roleId", roleId)
	}
	resp, err := a.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*TenantAssignment
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
