package client

import "context"

type GuestOperatingSystemClient struct {
	c *Client
}

func (c *Compute) GuestOperatingSystem() *GuestOperatingSystemClient {
	return &GuestOperatingSystemClient{c.c}
}

type GuestOperatingSystem struct {
	Moref    string
	Family   string
	FullName string
}

func (g *GuestOperatingSystemClient) List(
	ctx context.Context,
	machineManagerId string,
	hostClusterId string,
	hostId string,
	osFamily string) ([]*GuestOperatingSystem, error) {

	// TODO: filters
	r := g.c.newRequest("GET", "/api/compute/v1/vcenters/guest_operating_systems")
	r.params.Add("machineManagerId", machineManagerId)
	resp, err := g.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*GuestOperatingSystem
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (g *GuestOperatingSystemClient) Read(ctx context.Context, machineManagerId string, moref string) (*GuestOperatingSystem, error) {
	r := g.c.newRequest("GET", "/api/compute/v1/vcenters/guest_operating_systems/"+moref)
	r.params.Add("machineManagerId", machineManagerId)
	resp, err := g.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out GuestOperatingSystem
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
