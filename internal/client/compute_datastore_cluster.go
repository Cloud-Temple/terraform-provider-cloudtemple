package client

import "context"

type DatastoreClusterClient struct {
	c *Client
}

func (c *Compute) DatastoreCluster() *DatastoreClusterClient {
	return &DatastoreClusterClient{c.c}
}

type DatastoreCluster struct {
	ID               string
	Name             string
	Moref            string
	MachineManagerId string
	Datastores       []string
	Metrics          DatastoreClusterMetrics
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

func (d *DatastoreClusterClient) List(ctx context.Context, machineManagerId string, virtualDatacenterId string, hostId string, hostClusterId string) ([]*DatastoreCluster, error) {
	// TODO: filters
	r := d.c.newRequest("GET", "/api/compute/v1/vcenters/datastore_clusters")
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
	r := d.c.newRequest("GET", "/api/compute/v1/vcenters/datastore_clusters/"+id)
	resp, err := d.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out DatastoreCluster
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
