package client

import "context"

type OpenIaaSSnapshotClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) Snapshot() *OpenIaaSSnapshotClient {
	return &OpenIaaSSnapshotClient{c.c.c}
}

type OpenIaaSSnapshot struct {
	ID               string
	Description      string
	VirtualMachineID string
	Name             string
	CreateTime       int
}

func (v *OpenIaaSSnapshotClient) Read(ctx context.Context, id string) (*OpenIaaSSnapshot, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/snapshots/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSSnapshot
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type OpenIaaSSnapshotFilter struct {
	VirtualMachineID string `filter:"virtualMachineId"`
}

func (v *OpenIaaSSnapshotClient) List(ctx context.Context, filter *OpenIaaSSnapshotFilter) ([]*OpenIaaSSnapshot, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/snapshots")
	r.addFilter(filter)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*OpenIaaSSnapshot
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
