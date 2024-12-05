package client

type ComputeOpenIaaSClient struct {
	c *ComputeClient
}

func (c *ComputeClient) OpenIaaS() *ComputeOpenIaaSClient {
	return &ComputeOpenIaaSClient{c}
}
