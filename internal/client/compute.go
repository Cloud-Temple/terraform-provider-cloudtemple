package client

type ComputeClient struct {
	c *Client
}

func (c *Client) Compute() *ComputeClient {
	return &ComputeClient{c}
}
