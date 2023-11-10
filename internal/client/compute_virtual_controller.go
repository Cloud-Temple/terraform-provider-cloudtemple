package client

import "context"

type VirtualControllerClient struct {
	c *Client
}

func (c *ComputeClient) VirtualController() *VirtualControllerClient {
	return &VirtualControllerClient{c.c}
}

type VirtualController struct {
	ID               string   `terraform:"id"`
	VirtualMachineId string   `terraform:"virtual_machine_id"`
	HotAddRemove     bool     `terraform:"hot_add_remove"`
	Type             string   `terraform:"type"`
	SubType          string   `terraform:"sub_type"`
	Label            string   `terraform:"label"`
	Summary          string   `terraform:"summary"`
	VirtualDisks     []string `terraform:"virtual_disks"`
}

func (v *VirtualControllerClient) List(
	ctx context.Context,
	virtualMachineId string,
	types string) ([]*VirtualController, error) {

	// TODO: filters
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_controllers")
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

	var out []*VirtualController
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type CreateVirtualControllerRequest struct {
	VirtualMachineId string `json:"virtualMachineId"`
	Type             string `json:"type"`
	SubType          string `json:"subType,omitempty"`
}

func (n *VirtualControllerClient) Create(ctx context.Context, req *CreateVirtualControllerRequest) (string, error) {
	r := n.c.newRequest("POST", "/api/compute/v1/vcenters/virtual_controllers")
	r.obj = req
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualControllerClient) Read(ctx context.Context, id string) (*VirtualController, error) {
	r := v.c.newRequest("GET", "/api/compute/v1/vcenters/virtual_controllers/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out VirtualController
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type MountVirtualControllerRequest struct {
	ID                   string `json:"id"`
	IsoPath              string `json:"isoPath,omitempty"`
	ContentLibraryItemId string `json:"contentLibraryItemId,omitempty"`
}

func (n *VirtualControllerClient) Mount(ctx context.Context, req *MountVirtualControllerRequest) (string, error) {
	r := n.c.newRequest("PATCH", "/api/compute/v1/vcenters/virtual_controllers/cdrom/mount")
	r.obj = req
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualControllerClient) Unmount(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("PATCH", "/api/compute/v1/vcenters/virtual_controllers/cdrom/unmount")
	r.obj = map[string]string{"id": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualControllerClient) Connect(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("PATCH", "/api/compute/v1/vcenters/virtual_controllers/cdrom/connect")
	r.obj = map[string]string{"id": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualControllerClient) Disconnect(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("PATCH", "/api/compute/v1/vcenters/virtual_controllers/cdrom/disconnect")
	r.obj = map[string]string{"id": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualControllerClient) Delete(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("DELETE", "/api/compute/v1/vcenters/virtual_controllers/%s", id)
	return n.c.doRequestAndReturnActivity(ctx, r)
}
