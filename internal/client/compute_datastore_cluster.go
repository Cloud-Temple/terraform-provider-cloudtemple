package client

import "context"

type DatastoreClusterClient struct {
	c *Client
}

func (c *ComputeClient) DatastoreCluster() *DatastoreClusterClient {
	return &DatastoreClusterClient{c.c}
}

type DatastoreCluster struct {
	ID               string                  `terraform:"id"`
	Name             string                  `terraform:"name"`
	Moref            string                  `terraform:"moref"`
	MachineManagerId string                  `terraform:"machine_manager_id"`
	Datastores       []string                `terraform:"datastores"`
	Metrics          DatastoreClusterMetrics `terraform:"metrics"`
}

type DatastoreClusterMetrics struct {
	FreeCapacity                  int    `terraform:"free_capacity"`
	MaxCapacity                   int    `terraform:"max_capacity"`
	Enabled                       bool   `terraform:"enabled"`
	DefaultVmBehavior             string `terraform:"default_vm_behavior"`
	LoadBalanceInterval           int    `terraform:"load_balance_interval"`
	SpaceThresholdMode            string `terraform:"space_threshold_mode"`
	SpaceUtilizationThreshold     int    `terraform:"space_utilization_threshold"`
	MinSpaceUtilizationDifference int    `terraform:"min_space_utilization_difference"`
	ReservablePercentThreshold    int    `terraform:"reservable_percent_threshold"`
	ReservableThresholdMode       string `terraform:"reservable_threshold_mode"`
	IoLatencyThreshold            int    `terraform:"io_latency_threshold"`
	IoLoadImbalanceThreshold      int    `terraform:"io_load_imbalance_threshold"`
	IoLoadBalanceEnabled          bool   `terraform:"io_load_balance_enabled"`
}

func (d *DatastoreClusterClient) List(ctx context.Context, machineManagerId string, DatacenterId string, hostId string, hostClusterId string) ([]*DatastoreCluster, error) {
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
	r := d.c.newRequest("GET", "/api/compute/v1/vcenters/datastore_clusters/%s", id)
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
