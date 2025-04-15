package client

import "context"

type VirtualControllerClient struct {
	c *Client
}

func (c *ComputeClient) VirtualController() *VirtualControllerClient {
	return &VirtualControllerClient{c.c}
}

type VirtualController struct {
	ID               string
	VirtualMachineId string
	HotAddRemove     bool
	Type             string
	SubType          string
	Label            string
	Summary          string
	VirtualDisks     []string
}

type VirtualControllerFilter struct {
	VirtualMachineId string   `filter:"virtualMachineId"`
	Types            []string `filter:"types"`
}

func (v *VirtualControllerClient) List(ctx context.Context, filter *VirtualControllerFilter) ([]*VirtualController, error) {
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_controllers")
	r.addFilter(filter)
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
	r := n.c.newRequest("POST", "/compute/v1/vcenters/virtual_controllers")
	r.obj = req
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (v *VirtualControllerClient) Read(ctx context.Context, id string) (*VirtualController, error) {
	r := v.c.newRequest("GET", "/compute/v1/vcenters/virtual_controllers/%s", id)
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
	r := n.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_controllers/cdrom/mount")
	r.obj = req
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualControllerClient) Unmount(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_controllers/cdrom/unmount")
	r.obj = map[string]string{"id": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualControllerClient) Connect(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_controllers/cdrom/connect")
	r.obj = map[string]string{"id": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualControllerClient) Disconnect(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("PATCH", "/compute/v1/vcenters/virtual_controllers/cdrom/disconnect")
	r.obj = map[string]string{"id": id}
	return n.c.doRequestAndReturnActivity(ctx, r)
}

func (n *VirtualControllerClient) Delete(ctx context.Context, id string) (string, error) {
	r := n.c.newRequest("DELETE", "/compute/v1/vcenters/virtual_controllers/%s", id)
	return n.c.doRequestAndReturnActivity(ctx, r)
}
