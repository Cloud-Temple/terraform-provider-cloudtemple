package client

import "context"

type DatastoreClient struct {
	c *Client
}

func (c *ComputeClient) Datastore() *DatastoreClient {
	return &DatastoreClient{c.c}
}

type Datastore struct {
	ID                    string
	Name                  string
	MachineManager        BaseObject
	Moref                 string
	MaxCapacity           int
	FreeCapacity          int
	Accessible            int
	MaintenanceMode       bool
	UniqueId              string
	Type                  string
	VirtualMachinesNumber int
	HostsNumber           int
	HostsNames            []string
	AssociatedFolder      string
}

type DatastoreFilter struct {
	Name               string `filter:"name"`
	MachineManagerId   string `filter:"machineManagerId"`
	DatacenterId       string `filter:"datacenterId"`
	HostId             string `filter:"hostId"`
	HostClusterId      string `filter:"hostClusterId"`
	DatastoreClusterId string `filter:"datastoreClusterId"`
}

func (d *DatastoreClient) List(
	ctx context.Context,
	filter *DatastoreFilter) ([]*Datastore, error) {

	r := d.c.newRequest("GET", "/compute/v1/vcenters/datastores")
	r.addFilter(filter)
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
	r := d.c.newRequest("GET", "/compute/v1/vcenters/datastores/%s", id)
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
