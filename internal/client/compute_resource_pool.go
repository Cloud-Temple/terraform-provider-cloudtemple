package client

import "context"

type ResourcePoolClient struct {
	c *Client
}

func (c *ComputeClient) ResourcePool() *ResourcePoolClient {
	return &ResourcePoolClient{c.c}
}

type ResourcePool struct {
	ID             string              `terraform:"id"`
	Name           string              `terraform:"name"`
	Moref          string              `terraform:"moref"`
	Parent         ResourcePoolParent  `terraform:"parent"`
	Metrics        ResourcePoolMetrics `terraform:"metrics"`
	MachineManager BaseObject          `terraform:"machine_manager" terraform_flatten:"machine_manager"`
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
