package client

import "context"

type SnapshotClient struct {
	c *Client
}

func (c *Compute) Snapshot() *SnapshotClient {
	return &SnapshotClient{c.c}
}

type Snapshot struct {
	ID               string
	VirtualMachineId string
	Name             string
	CreateTime       int
}

func (s *SnapshotClient) List(ctx context.Context, virtualMachineId string) ([]*Snapshot, error) {
	r := s.c.newRequest("GET", "/api/compute/v1/vcenters/snapshots")
	r.params.Add("virtualMachineId", virtualMachineId)
	resp, err := s.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Snapshot
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
