package client

import "context"

type BackupJobClient struct {
	c *Client
}

func (c *BackupClient) Job() *BackupJobClient {
	return &BackupJobClient{c.c}
}

type BackupJob struct {
	ID          string `terraform:"id"`
	Name        string `terraform:"name"`
	DisplayName string `terraform:"display_name"`
	Type        string `terraform:"type"`
	Status      string `terraform:"status"`
	PolicyId    string `terraform:"policy_id"`
}

func (c *BackupJobClient) List(ctx context.Context, filter *struct{}) ([]*BackupJob, error) {
	r := c.c.newRequest("GET", "/api/backup/v1/jobs")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BackupJob
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *BackupJobClient) Read(ctx context.Context, id string) (*BackupJob, error) {
	r := c.c.newRequest("GET", "/api/backup/v1/jobs/"+id)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out BackupJob
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
