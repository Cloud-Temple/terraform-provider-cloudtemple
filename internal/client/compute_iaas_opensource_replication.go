package client

type ComputeOpenIaaSReplicationClient struct {
	c *ComputeOpenIaaSClient
}

func (c *ComputeOpenIaaSClient) Replication() *ComputeOpenIaaSReplicationClient {
	return &ComputeOpenIaaSReplicationClient{c}
}
