package client

import "context"

type ACLEntryClient struct {
	c *Client
}

func (c *ObjectStorage) ACLEntry() *ACLEntryClient {
	return &ACLEntryClient{c.c}
}

func (c *ACLEntryClient) Grant(ctx context.Context, bucket, role, storageAccount string) (string, error) {
	r := c.c.newRequest("POST", "/object-storage/v1/buckets/%s/grant/%s/to/%s", bucket, role, storageAccount)
	return c.c.doRequestAndReturnActivity(ctx, r)
}

func (c *ACLEntryClient) Revoke(ctx context.Context, bucket, role, storageAccount string) (string, error) {
	r := c.c.newRequest("DELETE", "/object-storage/v1/buckets/%s/revoke/%s/to/%s", bucket, role, storageAccount)
	return c.c.doRequestAndReturnActivity(ctx, r)
}
