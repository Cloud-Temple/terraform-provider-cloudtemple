package client

import "context"

type BackupJobSessionClient struct {
	c *Client
}

func (c *BackupClient) JobSession() *BackupJobSessionClient {
	return &BackupJobSessionClient{c.c}
}

type BackupJobSession struct {
	ID            string                 `terraform:"id"`
	JobName       string                 `terraform:"job_name"`
	SlaPolicyType string                 `terraform:"sla_policy_type"`
	JobId         string                 `terraform:"job_id"`
	Type          string                 `terraform:"type"`
	Duration      int                    `terraform:"duration"`
	Start         int                    `terraform:"start"`
	End           int                    `terraform:"end"`
	Status        string                 `terraform:"status"`
	Statistics    BackupStatistics       `terraform:"statistics"`
	SLAPolicies   []*BackupSLAPolicyStub `terraform:"sla_policies"`
}

type BackupStatistics struct {
	Total   int `terraform:"total"`
	Success int `terraform:"success"`
	Failed  int `terraform:"failed"`
	Skipped int `terraform:"skipped"`
}

type BackupSLAPolicyStub struct {
	ID   string `terraform:"id"`
	Name string `terraform:"name"`
	HREF string `terraform:"href"`
}

func (c *BackupJobSessionClient) List(ctx context.Context, filter *struct{}) ([]*BackupJobSession, error) {
	r := c.c.newRequest("GET", "/backup/v1/jobs/session")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*BackupJobSession
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}
