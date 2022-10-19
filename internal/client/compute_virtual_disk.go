package client

import "context"

type VirtualDiskClient struct {
	c *Client
}

func (c *Compute) VirtualDisk() *VirtualDiskClient {
	return &VirtualDiskClient{c.c}
}

type VirtualDisk struct {
	ID                  string
	VirtualMachineId    string
	MachineManagerId    string
	Name                string
	Capacity            int
	DiskUnitNumber      int
	ControllerBusNumber int
	DatastoreId         string
	DatastoreName       string
	InstantAccess       bool
	NativeId            string
	DiskPath            string
	ProvisioningType    string
	DiskMode            string
	Editable            bool
}

func (v *VirtualDiskClient) List(ctx context.Context, virtualMachineId string) ([]*VirtualDisk, error) {
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_disks")
	r.params.Add("virtualMachineId", virtualMachineId)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*VirtualDisk
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (v *VirtualDiskClient) Read(ctx context.Context, id string) (*VirtualDisk, error) {
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_disks/"+id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out VirtualDisk
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
