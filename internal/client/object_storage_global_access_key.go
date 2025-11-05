package client

import "context"

type GlobalAccessKeyClient struct {
	c *Client
}

func (c *ObjectStorage) GlobalAccessKey() *GlobalAccessKeyClient {
	return &GlobalAccessKeyClient{c.c}
}

type RenewAccessKeyResponse struct {
	AccessKeyID     string
	AccessSecretKey string
}

func (c *GlobalAccessKeyClient) Renew(ctx context.Context) (*RenewAccessKeyResponse, error) {
	r := c.c.newRequest("POST", "/storage/object/v1/namespaces/access_key/renew")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out RenewAccessKeyResponse
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
