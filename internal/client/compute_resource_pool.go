package client

import "context"

type ResourcePoolClient struct {
	c *Client
}

func (c *Compute) ResourcePool() *ResourcePoolClient {
	return &ResourcePoolClient{c.c}
}

type ResourcePool struct {
	ID               string
	Name             string
	MachineManagerID string
	Moref            string
	Parent           ResourcePoolParent
	Metrics          ResourcePoolMetrics
}

type ResourcePoolParent struct {
	ID   string
	Type string
}

type ResourcePoolMetrics struct {
	CPU    ResourcePoolCPUMetrics
	Memory ResourcePoolMemoryMetrics
}

type ResourcePoolCPUMetrics struct {
	MaxUsage        int
	ReservationUsed int
}

type ResourcePoolMemoryMetrics struct {
	MaxUsage        int
	ReservationUsed int
	BalloonedMemory int
}

func (rp *ResourcePoolClient) List(
	ctx context.Context,
	machineManagerID string,
	virtualDatacenterID string,
	hostClusterID string) ([]*ResourcePool, error) {

	// TODO: filters
	r := rp.c.newRequest("GET", "/api/compute/v1/vcenters/resource_pools")
	resp, err := rp.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*ResourcePool
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (rp *ResourcePoolClient) Read(ctx context.Context, id string) (*ResourcePool, error) {
	r := rp.c.newRequest("GET", "/api/compute/v1/vcenters/resource_pools/"+id)
	resp, err := rp.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out ResourcePool
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
