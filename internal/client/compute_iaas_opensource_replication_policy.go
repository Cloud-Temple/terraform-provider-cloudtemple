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
	// Start and End are Unix timestamps in MILLISECONDS (e.g. 1759830000000),
	// documented as int64 by the Compute API. They MUST NOT be `int`: the 386
	// and arm artifacts published by goreleaser build `int` as 32 bits, where
	// encoding/json refuses the value outright ("cannot unmarshal number
	// 1759830000000 into Go struct field ... of type int") and fails every read
	// that decodes this struct — including the Read of an Open IaaS virtual
	// machine, which resolves its replication policy through
	// /replication/virtual_machines/{id}/configurations (#543).
	LastRun struct {
		Start  int64
		End    int64
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
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out OpenIaaSReplicationPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// ListStrict behaves like List but requires a complete 200 answer: callers
// using the listing as EVIDENCE for state-shrinking decisions must fail
// closed on access-denied or partial answers (#275 doctrine, FF-5).
func (v *ComputeOpenIaaSReplicationPolicyClient) ListStrict(ctx context.Context) ([]*OpenIaaSReplicationPolicy, error) {
	r := v.c.newRequest("GET", "/compute/v1/open_iaas/replication/configurations")
	resp, err := v.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	// Strictly 200: a 206 partial listing cannot prove an absence.
	if err := requireHttpCodes(resp, 200); err != nil {
		return nil, err
	}

	var out []*OpenIaaSReplicationPolicy
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
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
