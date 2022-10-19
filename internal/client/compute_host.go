package client

import "context"

type HostClient struct {
	c *Client
}

func (c *Compute) Host() *HostClient {
	return &HostClient{c.c}
}

type Host struct {
	ID               string
	Name             string
	Moref            string
	MachineManagerID string
	Metrics          HostMetrics
	VirtualMachines  []HostVirtualMachinesStub
}

type HostMetrics struct {
	ESX               HostMetricsESXStub
	CPU               HostMetricsCPUStub
	Memory            HostMetricsMemoryStub
	MaintenanceStatus bool
	Uptime            int
	Connected         bool
}

type HostMetricsESXStub struct {
	Version  string
	Build    int
	FullName string
}

type HostMetricsCPUStub struct {
	OverallCPUUsage int
	CPUMhz          int
	CPUCores        int
	CPUThreads      int
}

type HostMetricsMemoryStub struct {
	MemorySize  int
	MemoryUsage int
}

type HostVirtualMachinesStub struct {
	ID   string
	Type string
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
	r := h.c.newRequest("GET", "/api/compute/v1/vcenters/hosts/"+id)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out Host
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
