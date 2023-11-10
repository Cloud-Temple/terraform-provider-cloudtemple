package client

import "context"

type SnapshotClient struct {
	c *Client
}

func (c *ComputeClient) Snapshot() *SnapshotClient {
	return &SnapshotClient{c.c}
}

type Snapshot struct {
	ID               string `terraform:"id"`
	VirtualMachineId string `terraform:"virtual_machine_id"`
	Name             string `terraform:"name"`
	CreateTime       int    `terraform:"create_time"`
}

func (s *SnapshotClient) List(ctx context.Context, virtualMachineId string) ([]*Snapshot, error) {
	r := s.c.newRequest("GET", "/compute/v1/vcenters/snapshots")
	r.params.Add("virtualMachineId", virtualMachineId)
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out []*Snapshot
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
