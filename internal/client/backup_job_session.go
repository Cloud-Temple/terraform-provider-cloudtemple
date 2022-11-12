package client

import "context"

type BackupJobSessionClient struct {
	c *Client
}

func (c *BackupClient) JobSession() *BackupJobSessionClient {
	return &BackupJobSessionClient{c.c}
}

type BackupJobSession struct {
	ID            string
	JobName       string
	SlaPolicyType string
	JobId         string
	Type          string
	Duration      int
	Start         int
	End           int
	Status        string
	Statistics    BackupStatistics
	SLAPolicies   []*BackupSLAPolicyStub
}

type BackupStatistics struct {
	Total   int
	Success int
	Failed  int
	Skipped int
}

type BackupSLAPolicyStub struct {
	ID   string
	Name string
	HREF string
}

func (c *BackupJobSessionClient) List(ctx context.Context, filter *struct{}) ([]*BackupJobSession, error) {
	r := c.c.newRequest("GET", "/api/backup/v1/jobs/session")
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
