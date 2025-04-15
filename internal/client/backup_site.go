package client

import "context"

type BackupSiteClient struct {
	c *Client
}

func (c *BackupClient) Site() *BackupSiteClient {
	return &BackupSiteClient{c.c}
}

type BackupSite struct {
	ID   string
	Name string
}

func (c *BackupSiteClient) List(ctx context.Context) ([]*BackupSite, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/sites")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BackupSite
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
