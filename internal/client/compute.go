package client

type Compute struct {
	c *Client
}

func (c *Client) Compute() *Compute {
	return &Compute{c}
}
