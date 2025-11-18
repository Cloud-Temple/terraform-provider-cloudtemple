package client

type ObjectStorage struct {
	c *Client
}

func (c *Client) ObjectStorage() *ObjectStorage {
	return &ObjectStorage{c}
}
