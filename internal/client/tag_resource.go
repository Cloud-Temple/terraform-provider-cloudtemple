package client

import "context"

type TagResourceClient struct {
	c *Client
}

func (c *TagClient) Resource() *TagResourceClient {
	return &TagResourceClient{c.c}
}

type CreateTagRequest struct {
	Key       string                      `json:"key"`
	Value     string                      `json:"value"`
	Resources []*CreateTagRequestResource `json:"resources"`
}

type CreateTagRequestResource struct {
	UUID   string `json:"uuid"`
	Type   string `json:"type"`
	Source string `json:"source"`
}

func (c *TagResourceClient) Create(ctx context.Context, req *CreateTagRequest) error {
	r := c.c.newRequest("POST", "/tag/v1/tags")
	r.obj = req
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp)
	return requireOK(resp)
}

func (c *TagResourceClient) Read(ctx context.Context, resourceId string) ([]*Tag, error) {
	r := c.c.newRequest("GET", "/tag/v1/tags/resources/%s", resourceId)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Tag
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *TagResourceClient) Delete(ctx context.Context, resourceId string, key string) error {
	r := c.c.newRequest("DELETE", "/tag/v1/tags/resources/%s/keys/%s", resourceId, key)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return err
	}
	defer closeResponseBody(resp)
	return requireHttpCodes(resp, 204)
}
