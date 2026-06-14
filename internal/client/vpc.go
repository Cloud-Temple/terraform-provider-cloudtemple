package client

// VPCClient is the entry point for the Shiva VPC API (base path /vpc/v1).
// It groups the read-only sub-clients used by the VPC datasources.
type VPCClient struct {
	c *Client
}

// VPC returns the VPC sub-client root.
func (c *Client) VPC() *VPCClient {
	return &VPCClient{c}
}

// VPC returns the client for the VPC resource itself.
func (v *VPCClient) VPC() *VPCVPCClient {
	return &VPCVPCClient{v.c}
}

// PrivateNetwork returns the client for VPC private networks.
func (v *VPCClient) PrivateNetwork() *VPCPrivateNetworkClient {
	return &VPCPrivateNetworkClient{v.c}
}

// StaticIP returns the client for VPC static IPs.
func (v *VPCClient) StaticIP() *VPCStaticIPClient {
	return &VPCStaticIPClient{v.c}
}

// FloatingIP returns the client for VPC floating IPs.
func (v *VPCClient) FloatingIP() *VPCFloatingIPClient {
	return &VPCFloatingIPClient{v.c}
}
