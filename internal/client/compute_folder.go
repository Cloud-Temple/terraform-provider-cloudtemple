package client

import "context"

type FolderClient struct {
	c *Client
}

func (c *Compute) Folder() *FolderClient {
	return &FolderClient{c.c}
}

type Folder struct {
	ID               string
	Name             string
	MachineManagerId string
}

func (f *FolderClient) List(
	ctx context.Context,
	machineManagerId string,
	virtualDatacenterId string) ([]*Folder, error) {

	// TODO: filters
	r := f.c.newRequest("GET", "/api/compute/v1/vcenters/folders")
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Folder
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (f *FolderClient) Read(ctx context.Context, id string) (*Folder, error) {
	r := f.c.newRequest("GET", "/api/compute/v1/vcenters/folders/"+id)
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out Folder
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}