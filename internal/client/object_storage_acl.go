package client

import "context"

type ACLClient struct {
	c *Client
}

func (c *ObjectStorage) ACL() *ACLClient {
	return &ACLClient{c.c}
}

type ACL struct {
	ID   string
	Name string
	Role string
}

// ListByBucket récupère tous les storage accounts qui ont accès à un bucket spécifique
func (c *ACLClient) ListByBucket(ctx context.Context, bucketName string) ([]*ACL, error) {
	r := c.c.newRequest("GET", "/object-storage/v1/buckets/%s/storage_accounts", bucketName)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*ACL
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// ListByStorageAccount récupère tous les buckets auxquels un storage account a accès
func (c *ACLClient) ListByStorageAccount(ctx context.Context, storageAccountName string) ([]*ACL, error) {
	r := c.c.newRequest("GET", "/object-storage/v1/storage_accounts/%s/buckets", storageAccountName)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*ACL
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
