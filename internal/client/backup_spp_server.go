package client

import "context"

type BackupSPPServerClient struct {
	c *Client
}

func (c *BackupClient) SPPServer() *BackupSPPServerClient {
	return &BackupSPPServerClient{c.c}
}

type BackupSPPServer struct {
	ID      string
	Name    string
	Address string
}

func (c *BackupSPPServerClient) List(ctx context.Context, tenantId string) ([]*BackupSPPServer, error) {
	r := c.c.newRequest("GET", "/api/backup/v1/spp_servers")
	r.params.Add("tenantId", tenantId)
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
	r := c.c.newRequest("GET", "/api/backup/v1/spp_servers/"+id)
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
