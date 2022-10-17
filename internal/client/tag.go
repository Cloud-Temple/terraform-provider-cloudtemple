package client

type TagClient struct {
	c *Client
}

func (c *Client) Tag() *TagClient {
	return &TagClient{c}
}

type Tag struct {
	Key      string
	Value    string
	Tenant   string
	Resource string
}
