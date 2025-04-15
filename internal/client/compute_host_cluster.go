package client

import "context"

type HostClusterClient struct {
	c *Client
}

func (c *ComputeClient) HostCluster() *HostClusterClient {
	return &HostClusterClient{c.c}
}

type HostCluster struct {
	ID                    string
	Name                  string
	Moref                 string
	Hosts                 []HostClusterHostStub
	Metrics               HostClusterMetrics
	VirtualMachinesNumber int
	MachineManager        BaseObject
	Datacenter            BaseObject
}

type HostClusterFilter struct {
	Name               string `filter:"name"`
	MachineManagerId   string `filter:"machineManagerId"`
	DatacenterId       string `filter:"datacenterId"`
	DatastoreId        string `filter:"datastoreId"`
	DatastoreClusterId string `filter:"datastoreClusterId"`
}

type HostClusterHostStub struct {
	ID   string
	Type string
}

type HostClusterMetrics struct {
	TotalCpu     int
	TotalMemory  int
	TotalStorage int
	CpuUsed      int
	MemoryUsed   int
	StorageUsed  int
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
