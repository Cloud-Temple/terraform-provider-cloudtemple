package client

import "context"

type BackupSLAPolicyClient struct {
	c *Client
}

func (c *BackupClient) SLAPolicy() *BackupSLAPolicyClient {
	return &BackupSLAPolicyClient{c.c}
}

type BackupSLAPolicy struct {
	ID          string
	Name        string
	SubPolicies []*BackupSLASubPolicy
}

type BackupSLASubPolicy struct {
	Type          string
	UseEncryption bool
	Software      bool
	Site          string
	Retention     BackupSLAPolicyRetention
	Trigger       BackupSLAPolicyTrigger
	Target        BackupSLAPolicyTarget
}

type BackupSLAPolicyTarget struct {
	ID           string
	Href         string
	ResourceType string
}

type BackupSLAPolicyRetention struct {
	Age int
}

type BackupSLAPolicyTrigger struct {
	Frequency    int
	Type         string
	ActivateDate int
}

type BackupSLAPolicyFilter struct {
	VirtualMachineId string `filter:"virtualMachineId"`
	VirtualDiskId    string `filter:"virtualDiskId"`
}

func (c *BackupSLAPolicyClient) List(ctx context.Context, filter *BackupSLAPolicyFilter) ([]*BackupSLAPolicy, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/policies")
	r.addFilter(filter)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BackupSLAPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *BackupSLAPolicyClient) Read(ctx context.Context, id string) (*BackupSLAPolicy, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/policies/%s", id)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out BackupSLAPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type BackupAssignVirtualMachineRequest struct {
	VirtualMachineIds []string `json:"virtualMachineIds"`
	SLAPolicies       []string `json:"slaPolicies"`
}

func (c *BackupSLAPolicyClient) AssignVirtualMachine(ctx context.Context, req *BackupAssignVirtualMachineRequest) (string, error) {
	r := c.c.newRequest("POST", "/backup/v1/spp/policies/assign/virtual_machine")
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}

type BackupAssignVirtualDiskRequest struct {
	VirtualDiskId string   `json:"virtualDiskId"`
	SLAPolicies   []string `json:"slaPolicies"`
}

func (c *BackupSLAPolicyClient) AssignVirtualDisk(ctx context.Context, req *BackupAssignVirtualDiskRequest) (string, error) {
	r := c.c.newRequest("POST", "/backup/v1/spp/policies/assign/virtual_disk")
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}
