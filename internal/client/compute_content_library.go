package client

import "context"

type ContentLibraryClient struct {
	c *Client
}

func (c *Compute) ContentLibrary() *ContentLibraryClient {
	return &ContentLibraryClient{c.c}
}

type ContentLibrary struct {
	ID               string
	Name             string
	MachineManagerID string
	Type             string
	Datastore        DatastoreLink
}

type DatastoreLink struct {
	ID   string
	Name string
}

func (c *ContentLibraryClient) List(ctx context.Context, machineManagerID string, datacenterID string, hostID string) ([]*ContentLibrary, error) {
	// TODO: filters
	r := c.c.newRequest("GET", "/api/compute/v1/vcenters/content_libraries")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*ContentLibrary
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *ContentLibraryClient) Read(ctx context.Context, id string) (*ContentLibrary, error) {
	r := c.c.newRequest("GET", "/api/compute/v1/vcenters/content_libraries/"+id)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out ContentLibrary
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
