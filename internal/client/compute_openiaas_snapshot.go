package client

import "context"

type OpenIaaSSnapshotClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) Snapshot() *OpenIaaSSnapshotClient {
	return &OpenIaaSSnapshotClient{c.c.c}
}

type OpenIaaSSnapshot struct {
	ID               string `terraform:"id"`
	Description      string `terraform:"description"`
	VirtualMachineID string `terraform:"virtual_machine_id"`
	Name             string `terraform:"name"`
	CreateTime       int    `terraform:"create_time"`
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

func (v *OpenIaaSSnapshotClient) List(ctx context.Context, virtualMachineId string) ([]*OpenIaaSSnapshot, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/snapshots")
	r.params.Add("virtualMachineId", virtualMachineId)
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
