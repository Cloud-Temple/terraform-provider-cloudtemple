package client

import (
	"context"
	"fmt"
	"time"

	"github.com/sethvargo/go-retry"
)

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

type BackupJobFilter struct {
	Type string
}

func (c *BackupJobClient) List(ctx context.Context, filter *BackupJobFilter) ([]*BackupJob, error) {
	r := c.c.newRequest("GET", "/api/backup/v1/jobs")
	if filter != nil && filter.Type != "" {
		r.params.Add("type", filter.Type)
	}
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
	r := c.c.newRequest("GET", "/api/backup/v1/jobs/%s", id)
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

type BackupJobRunRequest struct {
	PolicyId string `json:"policyId,omitempty"`
	JobId    string `json:"jobId"`
}

func (c *BackupJobClient) Run(ctx context.Context, req *BackupJobRunRequest) (string, error) {
	r := c.c.newRequest("POST", "/api/backup/v1/jobs/run")
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}

func (c *BackupJobClient) WaitForCompletion(ctx context.Context, id string) (*BackupJob, error) {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithCappedDuration(30*time.Second, b)

	var res *BackupJob
	var count int

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		count++
		job, err := c.Read(ctx, id)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("an error occured while getting job status: %s", err))
		}
		if job == nil {
			err := fmt.Errorf("the job %q could not be found", id)
			if count == 1 {
				return retry.RetryableError(err)
			}
			return err
		}
		res = job
		switch job.Status {
		case "IDLE":
			return nil
		case "RUNNING":
			return retry.RetryableError(fmt.Errorf("the job is running"))
		default:
			return fmt.Errorf("the job has failed: %v", job.Status)
		}
	})

	return res, err
}
