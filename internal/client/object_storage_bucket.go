package client

import "context"

type BucketClient struct {
	c *Client
}

func (c *ObjectStorage) Bucket() *BucketClient {
	return &BucketClient{c.c}
}

type Bucket struct {
	ID              string
	Name            string
	Namespace       string
	RetentionPeriod int64
	Versioning      string
	Endpoint        string
	TotalSize       string
	TotalSizeUnit   string
	TotalObjects    int64
	Tags            []struct {
		Key   string
		Value string
	}
	TotalObjectsDeleted string
	TotalSizeDeleted    string
}

func (c *BucketClient) List(ctx context.Context) ([]*Bucket, error) {
	r := c.c.newRequest("GET", "/object-storage/v1/buckets")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Bucket
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *BucketClient) Read(ctx context.Context, name string) (*Bucket, error) {
	r := c.c.newRequest("GET", "/object-storage/v1/buckets/%s", name)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out Bucket
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type CreateBucketRequest struct {
	Name       string   `json:"name"`
	AccessType string   `json:"accessType"`
	Whitelist  []string `json:"whitelist,omitempty"`
}

func (c *BucketClient) Create(ctx context.Context, req *CreateBucketRequest) (string, error) {
	r := c.c.newRequest("POST", "/object-storage/v1/buckets")
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}

func (c *BucketClient) Delete(ctx context.Context, name string) (string, error) {
	r := c.c.newRequest("DELETE", "/object-storage/v1/buckets/%s", name)
	return c.c.doRequestAndReturnActivity(ctx, r)
}

type UpdateWhitelistRequest struct {
	AccessType string   `json:"accessType"`
	Whitelist  []string `json:"whitelist,omitempty"`
}

func (c *BucketClient) UpdateWhitelist(ctx context.Context, name string, req *UpdateWhitelistRequest) (string, error) {
	r := c.c.newRequest("PUT", "/object-storage/v1/buckets/%s/whitelist", name)
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}

type UpdateVersioningRequest struct {
	Status string `json:"status"`
}

func (c *BucketClient) UpdateVersioning(ctx context.Context, name string, req *UpdateVersioningRequest) (string, error) {
	r := c.c.newRequest("PUT", "/object-storage/v1/buckets/%s/versioning", name)
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}
