package client

import "context"

type NamespaceClient struct {
	c *Client
}

func (c *ObjectStorage) Namespace() *NamespaceClient {
	return &NamespaceClient{c.c}
}

type Namespace struct {
	ID          string
	Name        string
	Region      string
	AccessKeyID string
}

func (c *NamespaceClient) Read(ctx context.Context) (*Namespace, error) {
	r := c.c.newRequest("GET", "/storage/object/v1/namespaces")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out Namespace
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
