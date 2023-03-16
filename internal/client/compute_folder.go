package client

import "context"

type FolderClient struct {
	c *Client
}

func (c *ComputeClient) Folder() *FolderClient {
	return &FolderClient{c.c}
}

type Folder struct {
	ID               string `terraform:"id"`
	Name             string `terraform:"name"`
	MachineManagerId string `terraform:"machine_manager_id"`
}

func (f *FolderClient) List(
	ctx context.Context,
	machineManagerId string,
	datacenterId string) ([]*Folder, error) {

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
	r := f.c.newRequest("GET", "/api/compute/v1/vcenters/folders/%s", id)
	resp, err := f.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out Folder
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
