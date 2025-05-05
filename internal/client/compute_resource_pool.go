package client

import "context"

type ResourcePoolClient struct {
	c *Client
}

func (c *ComputeClient) ResourcePool() *ResourcePoolClient {
	return &ResourcePoolClient{c.c}
}

type ResourcePool struct {
	ID             string
	Name           string
	Moref          string
	Parent         ResourcePoolParent
	Metrics        ResourcePoolMetrics
	MachineManager BaseObject
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

type ResourcePoolFilter struct {
	MachineManagerID string `filter:"machineManagerId"`
	DatacenterID     string `filter:"datacenterId"`
	HostClusterID    string `filter:"hostClusterId"`
}

func (rp *ResourcePoolClient) List(ctx context.Context, filter *ResourcePoolFilter) ([]*ResourcePool, error) {
	r := rp.c.newRequest("GET", "/compute/v1/vcenters/resource_pools")
	r.addFilter(filter)
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
	r := rp.c.newRequest("GET", "/compute/v1/vcenters/resource_pools/%s", id)
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
