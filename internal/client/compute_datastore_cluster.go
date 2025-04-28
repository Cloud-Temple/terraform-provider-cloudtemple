package client

import "context"

type DatastoreClusterClient struct {
	c *Client
}

func (c *ComputeClient) DatastoreCluster() *DatastoreClusterClient {
	return &DatastoreClusterClient{c.c}
}

type DatastoreCluster struct {
	ID             string
	Name           string
	Moref          string
	Datastores     []string
	Metrics        DatastoreClusterMetrics
	MachineManager BaseObject
	Datacenter     BaseObject
}

type DatastoreClusterFilter struct {
	Name             string `filter:"name"`
	MachineManagerId string `filter:"machineManagerId"`
	DatacenterId     string `filter:"datacenterId"`
	HostId           string `filter:"hostId"`
	HostClusterId    string `filter:"hostClusterId"`
}

type DatastoreClusterMetrics struct {
	FreeCapacity                  int
	MaxCapacity                   int
	Enabled                       bool
	DefaultVmBehavior             string
	LoadBalanceInterval           int
	SpaceThresholdMode            string
	SpaceUtilizationThreshold     int
	MinSpaceUtilizationDifference int
	ReservablePercentThreshold    int
	ReservableThresholdMode       string
	IoLatencyThreshold            int
	IoLoadImbalanceThreshold      int
	IoLoadBalanceEnabled          bool
}

func (d *DatastoreClusterClient) List(ctx context.Context, filter *DatastoreClusterFilter) ([]*DatastoreCluster, error) {
	r := d.c.newRequest("GET", "/compute/v1/vcenters/datastore_clusters")
	r.addFilter(filter)
	resp, err := d.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*DatastoreCluster
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (d *DatastoreClusterClient) Read(ctx context.Context, id string) (*DatastoreCluster, error) {
	r := d.c.newRequest("GET", "/compute/v1/vcenters/datastore_clusters/%s", id)
	resp, err := d.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out DatastoreCluster
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
