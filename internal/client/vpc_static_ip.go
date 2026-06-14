package client

import "context"

// VPCStaticIPClient handles read operations on VPC static IPs.
type VPCStaticIPClient struct {
	c *Client
}

// StaticIPFloatingIP is the floating IP bound to a static IP, as nested in the
// StaticIp schema. It carries the floating IP id and address; the API names the
// address field "ipAddress" here (distinct from FloatingIP.StaticIP which uses
// "address").
type StaticIPFloatingIP struct {
	ID        string `json:"id"`
	IPAddress string `json:"ipAddress"`
}

// StaticIP represents a static IP as returned by GET /vpc/v1/static_ips/{id},
// GET /vpc/v1/static_ips/mac/{mac} and
// GET /vpc/v1/private_networks/{id}/static_ips.
//
// virtualMachine, networkAdapter, resourceDescription and floatingIp are
// nullable in the API. source is one of xoa, vmware or custom.
type StaticIP struct {
	ID                  string              `json:"id"`
	IPAddress           string              `json:"ipAddress"`
	MacAddress          string              `json:"macAddress"`
	VirtualMachine      *BaseObject         `json:"virtualMachine"`
	NetworkAdapter      *BaseObject         `json:"networkAdapter"`
	Source              string              `json:"source"`
	ResourceDescription *string             `json:"resourceDescription"`
	FloatingIP          *StaticIPFloatingIP `json:"floatingIp"`
	VPC                 BaseObject          `json:"vpc"`
	PrivateNetwork      BaseObject          `json:"privateNetwork"`
}

// StaticIPFilter narrows a per-private-network static IP listing.
type StaticIPFilter struct {
	VirtualMachineID string `filter:"virtualMachineId"`
}

// List retrieves the static IP mappings of a private network, optionally
// filtered by virtual machine ID.
func (s *VPCStaticIPClient) List(ctx context.Context, privateNetworkID string, filter *StaticIPFilter) ([]*StaticIP, error) {
	r := s.c.newRequest("GET", "/vpc/v1/private_networks/%s/static_ips", privateNetworkID)
	r.addFilter(filter)
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*StaticIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read retrieves a single static IP by ID. It returns (nil, nil) when the
// static IP does not exist (403; the API returns 403 for an absent resource).
func (s *VPCStaticIPClient) Read(ctx context.Context, id string) (*StaticIP, error) {
	r := s.c.newRequest("GET", "/vpc/v1/static_ips/%s", id)
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out StaticIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// ReadByMAC retrieves a single static IP by MAC address. It returns (nil, nil)
// when no static IP matches the MAC (403; the API returns 403 for an absent resource).
func (s *VPCStaticIPClient) ReadByMAC(ctx context.Context, mac string) (*StaticIP, error) {
	r := s.c.newRequest("GET", "/vpc/v1/static_ips/mac/%s", mac)
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out StaticIP
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
