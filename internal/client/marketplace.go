package client

type MarketplaceClient struct {
	c *Client
}

func (c *Client) Marketplace() *MarketplaceClient {
	return &MarketplaceClient{c}
}
