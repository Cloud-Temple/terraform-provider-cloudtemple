package client

import "context"

type BackupOpenIaasPolicyClient struct {
	c *Client
}

func (c *BackupOpenIaasClient) Policy() *BackupOpenIaasPolicyClient {
	return &BackupOpenIaasPolicyClient{c.c.c}
}

type BackupOpenIaasPolicy struct {
	ID             string `terraform:"id"`
	Name           string `terraform:"name"`
	InternalID     string `terraform:"internal_id"`
	Running        bool   `terraform:"running"`
	Mode           string `terraform:"mode"`
	MachineManager struct {
		ID   string `terraform:"id"`
		Name string `terraform:"name"`
	} `terraform_flatten:"machine_manager"`
	Schedulers []struct {
		TemporarilyDisabled bool   `terraform:"temporarily_disabled"`
		Retention           int    `terraform:"retention"`
		Cron                string `terraform:"cron"`
		Timezone            string `terraform:"timezone"`
	} `terraform:"schedulers"`
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

func (v *BackupOpenIaasPolicyClient) List(ctx context.Context) ([]*BackupOpenIaasPolicy, error) {
	r := v.c.newRequest("GET", "/backup/v1/open_iaas/policies")
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
