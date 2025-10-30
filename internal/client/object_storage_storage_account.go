package client

import "context"

type StorageAccountClient struct {
	c *Client
}

func (c *ObjectStorage) StorageAccount() *StorageAccountClient {
	return &StorageAccountClient{c.c}
}

type StorageAccount struct {
	ID          string
	Name        string
	AccessKeyID string
	ARN         string
	CreateDate  string
	Path        string
	Tags        []struct {
		Key   string
		Value string
	}
}

func (c *StorageAccountClient) List(ctx context.Context) ([]*StorageAccount, error) {
	r := c.c.newRequest("GET", "/object-storage/v1/storage_accounts")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*StorageAccount
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *StorageAccountClient) Read(ctx context.Context, name string) (*StorageAccount, error) {
	r := c.c.newRequest("GET", "/object-storage/v1/storage_accounts/%s", name)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out StorageAccount
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
