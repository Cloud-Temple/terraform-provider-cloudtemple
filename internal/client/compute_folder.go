package client

import "context"

type FolderClient struct {
	c *Client
}

func (c *ComputeClient) Folder() *FolderClient {
	return &FolderClient{c.c}
}

type Folder struct {
	ID               string
	Name             string
	MachineManagerID string // DEPRECATED
	// MachineManager   BaseObject
}

type FolderFilter struct {
	Name             string `filter:"name"`
	MachineManagerID string `filter:"machineManagerId"`
	DatacenterID     string `filter:"datacenterId"`
}

func (f *FolderClient) List(ctx context.Context, filter *FolderFilter) ([]*Folder, error) {
	r := f.c.newRequest("GET", "/compute/v1/vcenters/folders")
	r.addFilter(filter)
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
	r := f.c.newRequest("GET", "/compute/v1/vcenters/folders/%s", id)
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
