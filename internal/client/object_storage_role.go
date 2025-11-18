package client

import "context"

type ObjectStorageRoleClient struct {
	c *Client
}

func (c *ObjectStorage) Role() *ObjectStorageRoleClient {
	return &ObjectStorageRoleClient{c.c}
}

type ObjectStorageRole struct {
	ID          string
	Name        string
	Description string
	Permissions []string
}

func (c *ObjectStorageRoleClient) List(ctx context.Context) ([]*ObjectStorageRole, error) {
	r := c.c.newRequest("GET", "/storage/object/v1/roles")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*ObjectStorageRole
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
