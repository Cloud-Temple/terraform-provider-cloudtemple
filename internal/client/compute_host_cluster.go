package client

import "context"

type HostClusterClient struct {
	c *Client
}

func (c *Compute) HostCluster() *HostClusterClient {
	return &HostClusterClient{c.c}
}

type HostCluster struct {
	ID                    string
	Name                  string
	Moref                 string
	Hosts                 []HostClusterHostStub
	Metrics               HostClusterMetrics
	VirtualMachinesNumber int
	MachineManagerId      string
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
	machineManagerId string,
	virtualDatacenterId string,
	datastoreId string) ([]*HostCluster, error) {

	// TODO: filters
	r := h.c.newRequest("GET", "/api/compute/v1/vcenters/host_clusters")
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
	r := h.c.newRequest("GET", "/api/compute/v1/vcenters/host_clusters/"+id)
	resp, err := h.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out HostCluster
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
