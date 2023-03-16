package client

import "context"

type ResourcePoolClient struct {
	c *Client
}

func (c *ComputeClient) ResourcePool() *ResourcePoolClient {
	return &ResourcePoolClient{c.c}
}

type ResourcePool struct {
	ID               string              `terraform:"id"`
	Name             string              `terraform:"name"`
	MachineManagerID string              `terraform:"machine_manager_id"`
	Moref            string              `terraform:"moref"`
	Parent           ResourcePoolParent  `terraform:"parent"`
	Metrics          ResourcePoolMetrics `terraform:"metrics"`
}

type ResourcePoolParent struct {
	ID   string `terraform:"id"`
	Type string `terraform:"type"`
}

type ResourcePoolMetrics struct {
	CPU    ResourcePoolCPUMetrics    `terraform:"cpu"`
	Memory ResourcePoolMemoryMetrics `terraform:"memory"`
}

type ResourcePoolCPUMetrics struct {
	MaxUsage        int `terraform:"max_usage"`
	ReservationUsed int `terraform:"reservation_used"`
}

type ResourcePoolMemoryMetrics struct {
	MaxUsage        int `terraform:"max_usage"`
	ReservationUsed int `terraform:"reservation_used"`
	BalloonedMemory int `terraform:"ballooned_memory"`
}

func (rp *ResourcePoolClient) List(
	ctx context.Context,
	machineManagerID string,
	DatacenterID string,
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
	r := rp.c.newRequest("GET", "/api/compute/v1/vcenters/resource_pools/%s", id)
	resp, err := rp.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out ResourcePool
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
