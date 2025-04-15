package client

import "context"

type BackupVCenterClient struct {
	c *Client
}

func (c *BackupClient) VCenter() *BackupVCenterClient {
	return &BackupVCenterClient{c.c}
}

type BackupVCenter struct {
	ID         string
	InternalId int
	InstanceId string
	Name       string
}

type BackupVCenterFilter struct {
	SppServerId string `filter:"sppServerId"`
}

func (c *BackupVCenterClient) List(ctx context.Context, filter *BackupVCenterFilter) ([]*BackupVCenter, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/vcenters")
	r.addFilter(filter)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BackupVCenter
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
