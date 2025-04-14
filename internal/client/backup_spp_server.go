package client

import "context"

type BackupSPPServerClient struct {
	c *Client
}

func (c *BackupClient) SPPServer() *BackupSPPServerClient {
	return &BackupSPPServerClient{c.c}
}

type BackupSPPServer struct {
	ID      string `terraform:"id"`
	Name    string `terraform:"name"`
	Address string `terraform:"address"`
}

type BackupSPPServerFilter struct {
	TenantId string `filter:"tenantId"`
}

func (c *BackupSPPServerClient) List(ctx context.Context, filter *BackupSPPServerFilter) ([]*BackupSPPServer, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/servers")
	r.addFilter(filter)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BackupSPPServer
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *BackupSPPServerClient) Read(ctx context.Context, id string) (*BackupSPPServer, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/servers/%s", id)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out BackupSPPServer
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
