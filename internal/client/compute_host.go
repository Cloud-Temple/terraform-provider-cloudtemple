package client

import "context"

type HostClient struct {
	c *Client
}

func (c *ComputeClient) Host() *HostClient {
	return &HostClient{c.c}
}

type Host struct {
	ID              string
	Name            string
	Moref           string
	MachineManager  BaseObject
	Metrics         HostMetrics
	VirtualMachines []HostVirtualMachinesStub
}

type HostMetrics struct {
	ESX             HostMetricsESXStub
	CPU             HostMetricsCPUStub
	Memory          HostMetricsMemoryStub
	MaintenanceMode bool
	Uptime          int
	Connected       bool
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

type HostFilter struct {
	Name             string `filter:"name"`
	MachineManagerID string `filter:"machineManagerId"`
	DatacenterID     string `filter:"datacenterId"`
	HostClusterID    string `filter:"hostClusterId"`
	DatastoreID      string `filter:"datastoreId"`
}

func (h *HostClient) List(ctx context.Context, filter *HostFilter) ([]*Host, error) {
	r := h.c.newRequest("GET", "/compute/v1/vcenters/hosts")
	r.addFilter(filter)
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
	r := h.c.newRequest("GET", "/compute/v1/vcenters/hosts/%s", id)
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
