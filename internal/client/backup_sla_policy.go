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
	Retention     BackupSLAPolicyRetention
	UseEncryption bool
	Software      bool
	Trigger       BackupSLAPolicyTrigger
	Site          string
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

func (c *BackupSLAPolicyClient) List(ctx context.Context, filters *struct{}) ([]*BackupSLAPolicy, error) {
	r := c.c.newRequest("GET", "/api/backup/v1/policies")
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
	r := c.c.newRequest("GET", "/api/backup/v1/policies/"+id)
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
