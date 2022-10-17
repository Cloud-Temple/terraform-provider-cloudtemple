package client

type IAM struct {
	c *Client
}

func (c *Client) IAM() *IAM {
	return &IAM{c}
}
