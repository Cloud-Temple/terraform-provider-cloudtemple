package client

import "context"

type BackupOpenIaasPolicyClient struct {
	c *Client
}

func (c *BackupOpenIaasClient) Policy() *BackupOpenIaasPolicyClient {
	return &BackupOpenIaasPolicyClient{c.c.c}
}

type BackupOpenIaasPolicy struct {
	ID              string
	Name            string
	InternalID      string
	Running         bool
	Mode            string
	MachineManager  BaseObject
	VirtualMachines []string
	Schedulers      []struct {
		TemporarilyDisabled bool
		Retention           int
		Cron                string
		Timezone            string
	}
}

func (v *BackupOpenIaasPolicyClient) Read(ctx context.Context, id string) (*BackupOpenIaasPolicy, error) {
	r := v.c.newRequest("GET", "/backup/v1/open_iaas/policies/%s", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out BackupOpenIaasPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type BackupOpenIaasPolicyFilter struct {
	Name             string `filter:"name"`
	MachineManagerId string `filter:"machineManagerId"`
	VirtualMachineId string `filter:"virtualMachineId"`
}

func (v *BackupOpenIaasPolicyClient) List(ctx context.Context, filter *BackupOpenIaasPolicyFilter) ([]*BackupOpenIaasPolicy, error) {
	r := v.c.newRequest("GET", "/backup/v1/open_iaas/policies")
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

	var out []*BackupOpenIaasPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

type BackupOpenIaasAssignPolicyRequest struct {
	VirtualMachineId string   `json:"virtualMachineId"`
	PolicyIds        []string `json:"policyIds"`
}

func (v *BackupOpenIaasPolicyClient) Assign(ctx context.Context, req *BackupOpenIaasAssignPolicyRequest) (string, error) {
	r := v.c.newRequest("POST", "/backup/v1/open_iaas/policies/assign")
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}
