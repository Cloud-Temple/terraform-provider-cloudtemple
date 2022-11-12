package client

import "context"

type BackupStorageClient struct {
	c *Client
}

func (c *BackupClient) Storage() *BackupStorageClient {
	return &BackupStorageClient{c.c}
}

type BackupStorage struct {
	ID               string
	ResourceType     string
	Type             string
	Site             string
	Name             string
	StorageId        string
	HostAddress      string
	PortNumber       int
	SSLConnection    bool
	InitializeStatus string
	Version          string
	IsReady          bool
	Capacity         BackupStorageCapacity
}

type BackupStorageCapacity struct {
	Free       int
	Total      int
	UpdateTime int
}

func (c *BackupStorageClient) List(ctx context.Context) ([]*BackupStorage, error) {
	r := c.c.newRequest("GET", "/api/backup/v1/storages")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BackupStorage
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
