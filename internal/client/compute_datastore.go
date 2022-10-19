package client

import "context"

type DatastoreClient struct {
	c *Client
}

func (c *Compute) Datastore() *DatastoreClient {
	return &DatastoreClient{c.c}
}

type Datastore struct {
	ID                    string
	Name                  string
	Moref                 string
	MaxCapacity           int
	FreeCapacity          int
	Accessible            int
	MaintenanceStatus     string
	UniqueId              string
	MachineManagerId      string
	Type                  string
	VirtualMachinesNumber int
	HostsNumber           int
	HostsNames            []string
	AssociatedFolder      string
}

func (d *DatastoreClient) List(
	ctx context.Context,
	machineManagerId string,
	virtualDatacenterId string,
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
	r := d.c.newRequest("GET", "/api/compute/v1/vcenters/datastores/"+id)
	resp, err := d.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out Datastore
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
