package client

import "context"

type BackupStorageClient struct {
	c *Client
}

func (c *BackupClient) Storage() *BackupStorageClient {
	return &BackupStorageClient{c.c}
}

type BackupStorage struct {
	ID               string                `terraform:"id"`
	ResourceType     string                `terraform:"resource_type"`
	Type             string                `terraform:"type"`
	Site             string                `terraform:"site"`
	Name             string                `terraform:"name"`
	StorageId        string                `terraform:"storage_id"`
	HostAddress      string                `terraform:"host_address"`
	PortNumber       int                   `terraform:"port_number"`
	SSLConnection    bool                  `terraform:"ssl_connection"`
	InitializeStatus string                `terraform:"initialize_status"`
	Version          string                `terraform:"version"`
	IsReady          bool                  `terraform:"is_ready"`
	Capacity         BackupStorageCapacity `terraform:"capacity"`
}

type BackupStorageCapacity struct {
	Free       int `terraform:"free"`
	Total      int `terraform:"total"`
	UpdateTime int `terraform:"update_time"`
}

func (c *BackupStorageClient) List(ctx context.Context) ([]*BackupStorage, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/storages")
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
