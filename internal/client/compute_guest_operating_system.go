package client

import "context"

type GuestOperatingSystemClient struct {
	c *Client
}

func (c *ComputeClient) GuestOperatingSystem() *GuestOperatingSystemClient {
	return &GuestOperatingSystemClient{c.c}
}

type GuestOperatingSystem struct {
	Moref    string `terraform:"moref"`
	Family   string `terraform:"family"`
	FullName string `terraform:"full_name"`
}

type GuestOperatingSystemFilter struct {
	HostID        string `filter:"hostId"`
	HostClusterID string `filter:"hostClusterId"`
	OsFamily      string `filter:"osFamily"`
	Version       string `filter:"version"`
}

func (g *GuestOperatingSystemClient) List(
	ctx context.Context,
	filter *GuestOperatingSystemFilter) ([]*GuestOperatingSystem, error) {

	r := g.c.newRequest("GET", "/compute/v1/vcenters/guest_operating_systems")
	r.addFilter(filter)
	resp, err := g.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*GuestOperatingSystem
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (g *GuestOperatingSystemClient) Read(ctx context.Context, moref string, filter *GuestOperatingSystemFilter) (*GuestOperatingSystem, error) {
	r := g.c.newRequest("GET", "/compute/v1/vcenters/guest_operating_systems/%s", moref)
	r.addFilter(filter)
	resp, err := g.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 500)
	if err != nil || !found {
		return nil, err
	}

	var out GuestOperatingSystem
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
