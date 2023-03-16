package client

import "context"

type DatastoreClient struct {
	c *Client
}

func (c *ComputeClient) Datastore() *DatastoreClient {
	return &DatastoreClient{c.c}
}

type Datastore struct {
	ID                    string   `terraform:"id"`
	Name                  string   `terraform:"name"`
	Moref                 string   `terraform:"moref"`
	MaxCapacity           int      `terraform:"max_capacity"`
	FreeCapacity          int      `terraform:"free_capacity"`
	Accessible            int      `terraform:"accessible"`
	MaintenanceStatus     string   `terraform:"maintenance_status"`
	UniqueId              string   `terraform:"unique_id"`
	MachineManagerId      string   `terraform:"machine_manager_id"`
	Type                  string   `terraform:"type"`
	VirtualMachinesNumber int      `terraform:"virtual_machines_number"`
	HostsNumber           int      `terraform:"hosts_number"`
	HostsNames            []string `terraform:"hosts_names"`
	AssociatedFolder      string   `terraform:"associated_folder"`
}

func (d *DatastoreClient) List(
	ctx context.Context,
	machineManagerId string,
	datacenterId string,
	hostId string,
	datastoreClusterId string,
	hostClusterId string) ([]*Datastore, error) {

	// TODO: filters
	r := d.c.newRequest("GET", "/api/compute/v1/vcenters/datastores")
	resp, err := d.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Datastore
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (d *DatastoreClient) Read(ctx context.Context, id string) (*Datastore, error) {
	r := d.c.newRequest("GET", "/api/compute/v1/vcenters/datastores/%s", id)
	resp, err := d.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out Datastore
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
