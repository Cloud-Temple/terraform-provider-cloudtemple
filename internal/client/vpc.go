package client

// VPC client for the Shiva /vpc/v1 API — UNDER ACTIVE REBUILD (v1.9.0).
//
// History: the platform replaced /vpc/v1 with breaking changes UNDER THE SAME URL
// (no /v2 coexistence). v1.8.0 removed the client-facing provider surface (the
// cloudtemple_vpc_* datasources and resources) and FROZE this client, to avoid
// shipping endpoints that would break server-side with no client recourse.
//
// v1.9.0 rebuilds VPC against the new contract, chunk by chunk: this client is the
// foundation, updated first (e.g. static IP create is now ASYNC — see
// vpc_static_ip.go CreateStart/WaitCreate), then the provider surface (Layer A) is
// restored on top in later chunks. So this package is NO LONGER FROZEN — changes
// here ARE the rebuild — but it is also not yet complete.
//
// SAFETY (kept until the rebuild is end-to-end validated): no provider surface
// references this client on the default path, and ct-validate exercises the /vpc/v1
// WRITE cycle ONLY via the explicit, opt-in "-cycles vpc -write" — never the blanket
// "-cycles all" (see vpcCycle.Quarantined in cmd/ct-validate/cycle_vpc.go). A routine
// sweep can therefore never fire VPC writes against this still-evolving contract.
//
// VPCClient is the entry point for the (rebuilding) /vpc/v1 API.
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
