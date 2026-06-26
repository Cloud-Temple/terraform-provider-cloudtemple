package client

// DEPRECATED CONTRACT / FROZEN — DO NOT EXTEND.
//
// This VPC client targets the Shiva VPC API at base path /vpc/v1, which is
// deprecated. A new contract is landing UNDER THE SAME /vpc/v1 URL with
// breaking changes (no /v2 coexistence), so this code cannot speak the new
// shape. The client-facing provider surface that consumed it (the
// cloudtemple_vpc_* datasources and resources) was removed from v1.8.0 to
// avoid shipping endpoints that will break server-side with no client
// recourse; the VPC features will be rebuilt against the new contract.
//
// This package is intentionally KEPT (it compiles and its tests run) as the
// foundation for that rebuild, but it is QUARANTINED: no provider surface
// references it, and ct-validate no longer exercises it in the default
// read-only path (only the opt-in, -write "vpc" cycle does). Treat any change
// here as part of the rebuild, not as maintenance of a live contract.
//
// VPCClient is the entry point for that (frozen) /vpc/v1 API.
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
