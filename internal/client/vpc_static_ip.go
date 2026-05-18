package client

import "context"

// VPCStaticIPClient handles static IP operations
type VPCStaticIPClient struct {
	c *Client
}

// StaticIP represents a static IP in a VPC
type StaticIP struct {
	ID                  string      `json:"id"`
	IPAddress           string      `json:"ipAddress"`
	MacAddress          string      `json:"macAddress"`
	VirtualMachine      *BaseObject `json:"virtualMachine"`
	NetworkAdapter      *BaseObject `json:"networkAdapter"`
	Source              string      `json:"source"`
	ResourceDescription *string     `json:"resourceDescription"`
	FloatingIP          *struct {
		ID        string `json:"id"`
		IPAddress string `json:"ipAddress"`
	} `json:"floatingIp"`
	VPC            BaseObject `json:"vpc"`
	PrivateNetwork BaseObject `json:"privateNetwork"`
}

// StaticIPFilter represents the filter for listing static IPs
type StaticIPFilter struct {
	VirtualMachineID string `filter:"virtualMachineId"`
}

// List retrieves all static IPs for a specific private network
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

// Read retrieves a specific static IP by ID
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
