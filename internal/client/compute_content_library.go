package client

import (
	"context"
	"time"
)

type ContentLibraryClient struct {
	c *Client
}

func (c *ComputeClient) ContentLibrary() *ContentLibraryClient {
	return &ContentLibraryClient{c.c}
}

type ContentLibrary struct {
	ID             string
	Name           string
	Type           string
	MachineManager BaseObject
	Datastore      BaseObject
}

type ContentLibraryFilter struct {
	Name             string `filter:"name"`
	MachineManagerId string `filter:"machineManagerId"`
}

func (c *ContentLibraryClient) List(ctx context.Context, filter *ContentLibraryFilter) ([]*ContentLibrary, error) {
	r := c.c.newRequest("GET", "/compute/v1/vcenters/content_libraries")
	r.addFilter(filter)
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
	r := c.c.newRequest("GET", "/compute/v1/vcenters/content_libraries/%s", id)
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
	Name             string
	Description      string
	Type             string
	CreationTime     time.Time
	Size             int
	Stored           bool
	LastModifiedTime string
	OvfProperties    []string
}

type ContentLibraryItemFilter struct {
	Name             string `filter:"name"`
	ContentLibraryId string
}

func (c *ContentLibraryClient) ListItems(ctx context.Context, filter *ContentLibraryItemFilter) ([]*ContentLibraryItem, error) {
	r := c.c.newRequest("GET", "/compute/v1/vcenters/content_libraries/%s/items", filter.ContentLibraryId)
	r.addFilter(filter)
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
	r := c.c.newRequest("GET", "/compute/v1/vcenters/content_libraries/%s/items/%s", contentLibraryId, contentLibraryItemId)
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

type ComputeContentLibraryItemDeployRequest struct {
	ContentLibraryId      string          `json:"-"`
	ContentLibraryItemId  string          `json:"contentLibraryItemId"`
	Name                  string          `json:"name"`
	HostClusterId         string          `json:"hostClusterId,omitempty"`
	HostId                string          `json:"hostId,omitempty"`
	DatastoreId           string          `json:"datastoreId,"`
	DatacenterId          string          `json:"datacenterId,omitempty"`
	PowerOn               bool            `json:"powerOn,omitempty"`
	DisksProvisioningType string          `json:"disksProvisioningType,omitempty"`
	DeployOptions         []*DeployOption `json:"deployOptions,omitempty"`
	NetworkData           []*NetworkData  `json:"networkData,omitempty"`
}

type DeployOption struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type NetworkData struct {
	NetworkAdapterId string `json:"networkAdapterId"`
	NetworkId        string `json:"networkId"`
}

func (c *ContentLibraryClient) Deploy(ctx context.Context, req *ComputeContentLibraryItemDeployRequest) (string, error) {
	r := c.c.newRequest("POST", "/compute/v1/vcenters/content_libraries/%s/items", req.ContentLibraryId)
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}
