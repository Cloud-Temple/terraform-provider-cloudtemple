package client

import "context"

type HostClient struct {
	c *Client
}

func (c *ComputeClient) Host() *HostClient {
	return &HostClient{c.c}
}

type Host struct {
	ID               string                    `terraform:"id"`
	Name             string                    `terraform:"name"`
	Moref            string                    `terraform:"moref"`
	MachineManagerID string                    `terraform:"machine_manager_id"`
	Metrics          HostMetrics               `terraform:"metrics"`
	VirtualMachines  []HostVirtualMachinesStub `terraform:"virtual_machines"`
}

type HostMetrics struct {
	ESX               HostMetricsESXStub    `terraform:"esx"`
	CPU               HostMetricsCPUStub    `terraform:"cpu"`
	Memory            HostMetricsMemoryStub `terraform:"memory"`
	MaintenanceStatus bool                  `terraform:"maintenance_status"`
	Uptime            int                   `terraform:"uptime"`
	Connected         bool                  `terraform:"connected"`
}

type HostMetricsESXStub struct {
	Version  string `terraform:"version"`
	Build    int    `terraform:"build"`
	FullName string `terraform:"full_name"`
}

type HostMetricsCPUStub struct {
	OverallCPUUsage int `terraform:"overall_cpu_usage"`
	CPUMhz          int `terraform:"cpu_mhz"`
	CPUCores        int `terraform:"cpu_cores"`
	CPUThreads      int `terraform:"cpu_threads"`
}

type HostMetricsMemoryStub struct {
	MemorySize  int `terraform:"memory_size"`
	MemoryUsage int `terraform:"memory_usage"`
}

type HostVirtualMachinesStub struct {
	ID   string `terraform:"id"`
	Type string `terraform:"type"`
}

func (h *HostClient) List(
	ctx context.Context,
	machineManagerID string,
	virtualDatacenterID string,
	hostClusterID string,
	datastoreID string) ([]*Host, error) {

	// TODO: filters
	r := h.c.newRequest("GET", "/api/compute/v1/vcenters/hosts")
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Host
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (h *HostClient) Read(ctx context.Context, id string) (*Host, error) {
	r := h.c.newRequest("GET", "/api/compute/v1/vcenters/hosts/%s", id)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out Host
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
