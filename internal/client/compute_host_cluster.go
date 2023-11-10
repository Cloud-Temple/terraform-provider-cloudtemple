package client

import "context"

type HostClusterClient struct {
	c *Client
}

func (c *ComputeClient) HostCluster() *HostClusterClient {
	return &HostClusterClient{c.c}
}

type HostCluster struct {
	ID                    string                `terraform:"id"`
	Name                  string                `terraform:"name"`
	Moref                 string                `terraform:"moref"`
	Hosts                 []HostClusterHostStub `terraform:"hosts"`
	Metrics               HostClusterMetrics    `terraform:"metrics"`
	VirtualMachinesNumber int                   `terraform:"virtual_machines_number"`
	MachineManagerId      string                `terraform:"machine_manager_id"`
}

type HostClusterFilter struct {
	Name             string `filter:"name"`
	MachineManagerId string `filter:"machineManagerId"`
	DatacenterId     string `filter:"datacenterId"`
	DatastoreId      string `filter:"datastoreId"`
}

type HostClusterHostStub struct {
	ID   string `terraform:"id"`
	Type string `terraform:"type"`
}

type HostClusterMetrics struct {
	TotalCpu     int `terraform:"total_cpu"`
	TotalMemory  int `terraform:"total_memory"`
	TotalStorage int `terraform:"total_storage"`
	CpuUsed      int `terraform:"cpu_used"`
	MemoryUsed   int `terraform:"memory_used"`
	StorageUsed  int `terraform:"storage_used"`
}

func (h *HostClusterClient) List(
	ctx context.Context,
	filter *HostClusterFilter) ([]*HostCluster, error) {

	// TODO: filters
	r := h.c.newRequest("GET", "/compute/v1/vcenters/host_clusters")
	r.addFilter(filter)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*HostCluster
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (h *HostClusterClient) Read(ctx context.Context, id string) (*HostCluster, error) {
	r := h.c.newRequest("GET", "/compute/v1/vcenters/host_clusters/%s", id)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out HostCluster
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
