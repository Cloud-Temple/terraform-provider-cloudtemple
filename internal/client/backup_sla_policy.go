package client

import "context"

type BackupSLAPolicyClient struct {
	c *Client
}

func (c *BackupClient) SLAPolicy() *BackupSLAPolicyClient {
	return &BackupSLAPolicyClient{c.c}
}

type BackupSLAPolicy struct {
	ID          string                `terraform:"id"`
	Name        string                `terraform:"name"`
	SubPolicies []*BackupSLASubPolicy `terraform:"sub_policies"`
}

type BackupSLASubPolicy struct {
	Type          string                   `terraform:"type"`
	UseEncryption bool                     `terraform:"use_encryption"`
	Software      bool                     `terraform:"software"`
	Site          string                   `terraform:"site"`
	Retention     BackupSLAPolicyRetention `terraform:"retention"`
	Trigger       BackupSLAPolicyTrigger   `terraform:"trigger"`
	Target        BackupSLAPolicyTarget    `terraform:"target"`
}

type BackupSLAPolicyTarget struct {
	ID           string `terraform:"id"`
	Href         string `terraform:"href"`
	ResourceType string `terraform:"resource_type"`
}

type BackupSLAPolicyRetention struct {
	Age int `terraform:"age"`
}

type BackupSLAPolicyTrigger struct {
	Frequency    int    `terraform:"frequency"`
	Type         string `terraform:"type"`
	ActivateDate int    `terraform:"activate_date"`
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
