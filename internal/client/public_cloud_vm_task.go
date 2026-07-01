package client

import (
	"context"
	"strconv"
)

type PublicCloudVMTaskClient struct {
	c *Client
}

// Task returns the task sub-client (read-only). Tasks are a DIAGNOSTIC object
// (upstream machine-manager / xo-server tasks); they are UNRELATED to the
// activities used to track writes and are never used to follow a write.
func (v *PublicCloudVMClient) Task() *PublicCloudVMTaskClient {
	return &PublicCloudVMTaskClient{v.c}
}

// PublicCloudVMTask mirrors an element of GET /vm_instances/v1/tasks (bare array,
// camelCase). Diagnostic only.
type PublicCloudVMTask struct {
	ID            string
	VmID          string
	TaskType      string
	Status        string
	Message       string
	FailureCode   string
	RetriedFromID string
	CreatedAt     string
	UpdatedAt     string
	CompletedAt   string
}

// List returns tasks. When vmID is non-empty the per-VM endpoint
// (/virtual_machines/{vmID}/tasks) is used; otherwise the global list
// (/v1/tasks). limit, when > 0, is passed through as the `limit` query param
// (max 500 global / 200 per-VM per the API). This is a diagnostic listing and is
// not auto-paginated.
func (t *PublicCloudVMTaskClient) List(ctx context.Context, vmID string, limit int) ([]*PublicCloudVMTask, error) {
	var req *request
	if vmID != "" {
		req = t.c.newRequest("GET", "/vm_instances/v1/virtual_machines/%s/tasks", vmID)
	} else {
		req = t.c.newRequest("GET", "/vm_instances/v1/tasks")
	}
	if limit > 0 {
		req.params.Add("limit", strconv.Itoa(limit))
	}
	resp, err := t.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*PublicCloudVMTask
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Read returns a single task by id. A positive 404 maps to (nil, nil); any other
// non-OK code (403, 5xx) fails closed with an error.
func (t *PublicCloudVMTaskClient) Read(ctx context.Context, id string) (*PublicCloudVMTask, error) {
	req := t.c.newRequest("GET", "/vm_instances/v1/tasks/%s", id)
	resp, err := t.c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out PublicCloudVMTask
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
