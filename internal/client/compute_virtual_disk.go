package client

import "context"

type VirtualDiskClient struct {
	c *Client
}

func (c *ComputeClient) VirtualDisk() *VirtualDiskClient {
	return &VirtualDiskClient{c.c}
}

type VirtualDisk struct {
	ID                  string     `terraform:"id"`
	VirtualMachineId    string     `terraform:"virtual_machine_id"`
	MachineManager      BaseObject `terraform_flatten:"machine_manager"`
	Name                string     `terraform:"name"`
	Capacity            int        `terraform:"capacity"`
	DiskUnitNumber      int        `terraform:"disk_unit_number"`
	ControllerBusNumber int        `terraform:"controller_bus_number"`
	DatastoreId         string     `terraform:"datastore_id"`
	DatastoreName       string     `terraform:"datastore_name"`
	InstantAccess       bool       `terraform:"instant_access"`
	NativeId            string     `terraform:"native_id"`
	DiskPath            string     `terraform:"disk_path"`
	ProvisioningType    string     `terraform:"provisioning_type"`
	DiskMode            string     `terraform:"disk_mode"`
	Editable            bool       `terraform:"editable"`
	Controller          struct {
		ID        string `terraform:"id"`
		BusNumber int    `terraform:"bus_number"`
		Type      string `terraform:"type"`
	} `terraform_flatten:"controller"`
}

func (v *VirtualDiskClient) List(ctx context.Context, virtualMachineId string) ([]*VirtualDisk, error) {
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_disks")
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

type CreateVirtualDiskRequest struct {
	ControllerId       string `json:"controllerId,omitempty"`
	ProvisioningType   string `json:"provisioningType"`
	DiskMode           string `json:"diskMode"`
	Capacity           int    `json:"capacity"`
	VirtualMachineId   string `json:"virtualMachineId"`
	DatastoreId        string `json:"datastoreId,omitempty"`
	DatastoreClusterId string `json:"datastoreClusterId,omitempty"`
}

func (n *VirtualDiskClient) Create(ctx context.Context, req *CreateVirtualDiskRequest) (string, error) {
	r := n.c.newRequest("POST", "/compute/v1/vcenters/virtual_disks")
	r.obj = req
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualDiskClient) Read(ctx context.Context, id string) (*VirtualDisk, error) {
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_disks/%s", id)
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

type UpdateVirtualDiskRequest struct {
	ID          string `json:"id"`
	NewCapacity int    `json:"newCapacity,omitempty"`
	DiskMode    string `json:"diskMode,omitempty"`
}

func (n *VirtualDiskClient) Update(ctx context.Context, req *UpdateVirtualDiskRequest) (string, error) {
	r := n.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_disks")
	r.obj = req
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualDiskClient) Delete(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("DELETE", "/compute/v1/vcenters/virtual_disks/%s", id)
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualDiskClient) Mount(ctx context.Context, virtualMachineId string, path string) (string, error) {
	r := n.c.newRequest("POST", "/compute/v1/vcenters/virtual_disks/mount")
	r.obj = map[string]string{
		"virtualMachineId": virtualMachineId,
		"path":             path,
	}
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualDiskClient) Unmount(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("POST", "/compute/v1/vcenters/virtual_disks/unmount")
	r.obj = map[string]string{"virtualDiskId": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}
