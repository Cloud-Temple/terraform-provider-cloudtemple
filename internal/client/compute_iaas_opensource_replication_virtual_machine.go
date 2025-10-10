package client

import "context"

type ComputeOpenIaaSReplicationPolicyVirtualMachineClient struct {
	c *Client
}

func (c *ComputeOpenIaaSReplicationPolicyClient) VirtualMachine() *ComputeOpenIaaSReplicationPolicyVirtualMachineClient {
	return &ComputeOpenIaaSReplicationPolicyVirtualMachineClient{c.c}
}

func (v *ComputeOpenIaaSReplicationPolicyVirtualMachineClient) Read(ctx context.Context, id string) (*OpenIaaSReplicationPolicy, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/replication/virtual_machines/%s/configurations", id)
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 403)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSReplicationPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

type AssociateReplicationPolicyToVirtualMachineRequest struct {
	ConfigurationID string `json:"configurationId"`
}

func (v *ComputeOpenIaaSReplicationPolicyVirtualMachineClient) Associate(ctx context.Context, virtualMachineId string, req *AssociateReplicationPolicyToVirtualMachineRequest) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/open_iaas/replication/virtual_machines/%s/configurations", virtualMachineId)
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *ComputeOpenIaaSReplicationPolicyVirtualMachineClient) Dissociate(ctx context.Context, virtualMachineId string) (string, error) {
	r := v.c.newRequest("DELETE", "/compute/v1/open_iaas/replication/virtual_machines/%s/configurations", virtualMachineId)
	return v.c.doRequestAndReturnActivity(ctx, r)
}
