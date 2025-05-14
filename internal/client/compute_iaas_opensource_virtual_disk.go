package client

import "context"

type OpenIaaSVirtualDiskClient struct {
	c *Client
}

func (c *ComputeOpenIaaSClient) VirtualDisk() *OpenIaaSVirtualDiskClient {
	return &OpenIaaSVirtualDiskClient{c.c.c}
}

type OpenIaaSVirtualDiskConnection struct {
	ID        string
	Name      string
	ReadOnly  bool
	Connected bool
}

type OpenIaaSVirtualDisk struct {
	ID                string
	InternalID        string
	Name              string
	Description       string
	Size              int
	Usage             int
	IsSnapshot        bool
	StorageRepository BaseObject
	VirtualMachines   []OpenIaaSVirtualDiskConnection
	Templates         []struct {
		ID       string
		Name     string
		ReadOnly bool
	}
}

type OpenIaaSVirtualDiskCreateRequest struct {
	Name                string `json:"name"`
	Size                int    `json:"size"`
	Mode                string `json:"mode"`
	StorageRepositoryID string `json:"storageRepositoryId"`
	VirtualMachineID    string `json:"virtualMachineId"`
	Bootable            bool   `json:"bootable"`
}

func (v *OpenIaaSVirtualDiskClient) Create(ctx context.Context, req *OpenIaaSVirtualDiskCreateRequest) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/open_iaas/virtual_disks")
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSVirtualDiskClient) Read(ctx context.Context, id string) (*OpenIaaSVirtualDisk, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/virtual_disks/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSVirtualDisk
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type OpenIaaSVirtualDiskFilter struct {
	VirtualMachineID    string `filter:"virtualMachineId"`
	TemplateID          string `filter:"templateId"`
	StorageRepositoryID string `filter:"storageRepositoryId"`
}

func (v *OpenIaaSVirtualDiskClient) List(ctx context.Context, filter *OpenIaaSVirtualDiskFilter) ([]*OpenIaaSVirtualDisk, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/virtual_disks")
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

	var out []*OpenIaaSVirtualDisk
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type OpenIaaSVirtualDiskUpdateRequest struct {
	ID   string // No JSON tag, as this is the URL parameter
	Name string `json:"name,omitempty"`
	Size int    `json:"size,omitempty"`
}

func (v *OpenIaaSVirtualDiskClient) Update(ctx context.Context, req *OpenIaaSVirtualDiskUpdateRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/virtual_disks/%s", req.ID)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

type OpenIaaSVirtualDiskRelocateRequest struct {
	ID                  string // No JSON tag, as this is the URL parameter
	StorageRepositoryID string `json:"storageRepositoryId"`
}

func (v *OpenIaaSVirtualDiskClient) Relocate(ctx context.Context, req *OpenIaaSVirtualDiskRelocateRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/virtual_disks/%s/relocate", req.ID)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

type OpenIaaSVirtualDiskConnectionRequest struct {
	ID               string // No JSON tag, as this is the URL parameter
	VirtualMachineID string `json:"virtualMachineId"`
}

func (v *OpenIaaSVirtualDiskClient) Connect(ctx context.Context, req *OpenIaaSVirtualDiskConnectionRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/virtual_disks/%s/connect", req.ID)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSVirtualDiskClient) Disconnect(ctx context.Context, req *OpenIaaSVirtualDiskConnectionRequest) (string, error) {
	r := v.c.newRequest("PATCH", "/compute/v1/open_iaas/virtual_disks/%s/disconnect", req.ID)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

type OpenIaaSVirtualDiskAttachRequest struct {
	VirtualMachineID string `json:"virtualMachineId"`
	Bootable         bool   `json:"bootable,omitempty"`
	Mode             string `json:"mode"`
	Position         string `json:"position,omitempty"`
}

func (v *OpenIaaSVirtualDiskClient) Attach(ctx context.Context, id string, req *OpenIaaSVirtualDiskAttachRequest) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/open_iaas/virtual_disks/%s/attach", id)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *OpenIaaSVirtualDiskClient) Delete(ctx context.Context, id string) (string, error) {
	r := v.c.newRequest("DELETE", "/compute/v1/open_iaas/virtual_disks/%s", id)
	return v.c.doRequestAndReturnActivity(ctx, r)
}
