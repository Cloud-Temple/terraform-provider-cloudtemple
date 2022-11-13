package client

import (
	"context"
	"time"
)

type ContentLibraryClient struct {
	c *Client
}

func (c *Compute) ContentLibrary() *ContentLibraryClient {
	return &ContentLibraryClient{c.c}
}

type ContentLibrary struct {
	ID               string        `terraform:"id"`
	Name             string        `terraform:"name"`
	MachineManagerID string        `terraform:"machine_manager_id"`
	Type             string        `terraform:"type"`
	Datastore        DatastoreLink `terraform:"datastore"`
}

type DatastoreLink struct {
	ID   string `terraform:"id"`
	Name string `terraform:"name"`
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
	r := c.c.newRequest("GET", "/api/compute/v1/vcenters/content_libraries/%s", id)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out ContentLibrary
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type ContentLibraryItem struct {
	ID               string
	ContentLibraryId string
	Name             string
	Description      string
	Type             string
	CreationTime     time.Time
	Size             int
	Stored           bool
	LastModifiedTime string
	OvfProperties    []string
}

func (c *ContentLibraryClient) ListItems(ctx context.Context, id string) ([]*ContentLibraryItem, error) {
	r := c.c.newRequest("GET", "/api/compute/v1/vcenters/content_libraries/%s/items", id)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*ContentLibraryItem
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *ContentLibraryClient) ReadItem(ctx context.Context, contentLibraryId, contentLibraryItemId string) (*ContentLibraryItem, error) {
	r := c.c.newRequest("GET", "/api/compute/v1/vcenters/content_libraries/%s/items/%s", contentLibraryId, contentLibraryItemId)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out ContentLibraryItem
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
