package client

// VPCClient provides access to VPC API endpoints
type VPCClient struct {
	c *Client
}

// VPC returns the VPC client
func (c *Client) VPC() *VPCClient {
	return &VPCClient{c}
}

// VPCClient returns the VPC VPC client
func (c *VPCClient) VPC() *VPCVPCClient {
	return &VPCVPCClient{c.c}
}

// PrivateNetwork returns the VPC private network client
func (c *VPCClient) PrivateNetwork() *VPCPrivateNetworkClient {
	return &VPCPrivateNetworkClient{c.c}
}

// StaticIP returns the VPC static IP client
func (c *VPCClient) StaticIP() *VPCStaticIPClient {
	return &VPCStaticIPClient{c.c}
}

// FloatingIP returns the VPC floating IP client
func (c *VPCClient) FloatingIP() *VPCFloatingIPClient {
	return &VPCFloatingIPClient{c.c}
}
