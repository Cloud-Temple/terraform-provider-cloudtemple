package client

import "context"

type BackupVCenterClient struct {
	c *Client
}

func (c *BackupClient) VCenter() *BackupVCenterClient {
	return &BackupVCenterClient{c.c}
}

type BackupVCenter struct {
	ID          string `terraform:"id"`
	InternalId  int    `terraform:"internal_id"`
	InstanceId  string `terraform:"instance_id"`
	SppServerId string `terraform:"spp_server_id"`
	Name        string `terraform:"name"`
}

func (c *BackupVCenterClient) List(ctx context.Context, sppServerId string) ([]*BackupVCenter, error) {
	r := c.c.newRequest("GET", "/backup/v1/vcenters")
	r.params.Add("sppServerId", sppServerId)
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
