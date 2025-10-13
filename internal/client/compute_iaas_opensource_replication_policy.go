package client

import (
	"context"
)

type ComputeOpenIaaSReplicationPolicyClient struct {
	c *Client
}

func (c *ComputeOpenIaaSReplicationClient) Policy() *ComputeOpenIaaSReplicationPolicyClient {
	return &ComputeOpenIaaSReplicationPolicyClient{c.c.c.c}
}

type OpenIaaSReplicationPolicy struct {
	ID                string
	Name              string
	StorageRepository struct {
		ID   string
		Name string
	}
	Pool struct {
		ID    string
		Name  string
		Label string
	}
	MachineManager struct {
		ID   string
		Name string
	}
	LastRun struct {
		Start  int
		End    int
		Status string
	}
	Interval struct {
		Hours   int
		Minutes int
	}
}

type ReplicationPolicyInterval struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minute,omitempty"`
}

type CreateOpenIaaSReplicationPolicyRequest struct {
	Name                string                    `json:"name"`
	StorageRepositoryID string                    `json:"storageRepositoryId"`
	Interval            ReplicationPolicyInterval `json:"interval"`
}

func (v *ComputeOpenIaaSReplicationPolicyClient) Create(ctx context.Context, req *CreateOpenIaaSReplicationPolicyRequest) (string, error) {
	r := v.c.newRequest("POST", "/compute/v1/open_iaas/replication/configurations")
	r.obj = req
	return v.c.doRequestAndReturnActivity(ctx, r)
}

func (v *ComputeOpenIaaSReplicationPolicyClient) Read(ctx context.Context, id string) (*OpenIaaSReplicationPolicy, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/replication/configurations/%s", id)
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

func (v *ComputeOpenIaaSReplicationPolicyClient) List(ctx context.Context) ([]*OpenIaaSReplicationPolicy, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/replication/configurations")
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*OpenIaaSReplicationPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (v *ComputeOpenIaaSReplicationPolicyClient) Delete(ctx context.Context, id string) (string, error) {
	r := v.c.newRequest("DELETE", "/compute/v1/open_iaas/replication/configurations/%s", id)
	return v.c.doRequestAndReturnActivity(ctx, r)
}
