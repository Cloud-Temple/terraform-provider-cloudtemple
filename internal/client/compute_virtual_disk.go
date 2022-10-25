package client

import "context"

type VirtualDiskClient struct {
	c *Client
}

func (c *Compute) VirtualDisk() *VirtualDiskClient {
	return &VirtualDiskClient{c.c}
}

type VirtualDisk struct {
	ID                  string `terraform:"id"`
	VirtualMachineId    string `terraform:"virtual_machine_id"`
	MachineManagerId    string `terraform:"machine_manager_id"`
	Name                string `terraform:"name"`
	Capacity            int    `terraform:"capacity"`
	DiskUnitNumber      int    `terraform:"disk_unit_number"`
	ControllerBusNumber int    `terraform:"controller_bus_number"`
	DatastoreId         string `terraform:"datastore_id"`
	DatastoreName       string `terraform:"datastore_name"`
	InstantAccess       bool   `terraform:"instant_access"`
	NativeId            string `terraform:"native_id"`
	DiskPath            string `terraform:"disk_path"`
	ProvisioningType    string `terraform:"provisioning_type"`
	DiskMode            string `terraform:"disk_mode"`
	Editable            bool   `terraform:"editable"`
}

func (v *VirtualDiskClient) List(ctx context.Context, virtualMachineId string) ([]*VirtualDisk, error) {
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_disks")
	r.params.Add("virtualMachineId", virtualMachineId)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
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
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out VirtualDisk
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
