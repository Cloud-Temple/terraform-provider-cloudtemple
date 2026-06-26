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
	ID          string
	Name        string
	DisplayName string
	Type        string
	Status      string
	PolicyId    string
}

type BackupJobFilter struct {
	Type string
}

func (c *BackupJobClient) List(ctx context.Context, filter *BackupJobFilter) ([]*BackupJob, error) {
	r := c.c.newRequest("GET", "/backup/v1/spp/jobs")
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
	r := c.c.newRequest("GET", "/backup/v1/spp/jobs/%s", id)
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
	JobId    string `json:"jobId"`
	PolicyId string `json:"policyId,omitempty"`
}

func (c *BackupJobClient) Run(ctx context.Context, req *BackupJobRunRequest) (string, error) {
	r := c.c.newRequest("POST", "/backup/v1/spp/jobs/run")
	r.obj = req
	return c.c.doRequestAndReturnActivity(ctx, r)
}

type BackupJobCompletionError struct {
	message string
	job     *BackupJob
}

const backupJobCompletionErrorMessage = `%s:

  Status: %s
  Name: %s
  Display name: %s
  Type: %s
  PolicyId: %q
`

func (b *BackupJobCompletionError) Error() string {
	if b.job == nil {
		return b.message
	}

	return fmt.Sprintf(
		backupJobCompletionErrorMessage,
		b.message,
		b.job.Status,
		b.job.Name,
		b.job.DisplayName,
		b.job.Type,
		b.job.PolicyId,
	)
}

func (c *BackupJobClient) WaitForCompletion(ctx context.Context, id string, options *WaiterOptions) (*BackupJob, error) {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithCappedDuration(30*time.Second, b)

	return waitForBackupJobCompletion(ctx, id, func(ctx context.Context) (*BackupJob, error) {
		return c.Read(ctx, id)
	}, b, options)
}

// backupJobReadFunc abstracts the job read so the polling loop can be unit
// tested without HTTP calls or real sleeps.
type backupJobReadFunc func(ctx context.Context) (*BackupJob, error)

// waitForBackupJobCompletion is the polling loop behind WaitForCompletion, with
// the read and the backoff injected (mirrors waitForActivityCompletion). Behavior
// is preserved exactly: transient read errors are retried, permanent/timeout/ctx
// errors fail at once, an IDLE job is completion, RUNNING keeps polling, anything
// else is a failure, and the not-found tolerance is `count == 1` ONLY — i.e. a nil
// job on the FIRST polling attempt; any prior attempt (a transient blip or a
// RUNNING status) consumes it (see #293 Finding F2; NOT changed here).
func waitForBackupJobCompletion(ctx context.Context, id string, read backupJobReadFunc, b retry.Backoff, options *WaiterOptions) (*BackupJob, error) {
	var res *BackupJob
	var count int

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		count++
		job, err := read(ctx)
		if err != nil {
			// Only a transient read error is worth retrying. A configured request
			// timeout, a context error or a permanent error (4xx, decode) must fail
			// the wait at once — retrying a timeout here would stall for minutes.
			if isTransientAPIError(err) {
				return options.retryableError(&BackupJobCompletionError{
					message: fmt.Sprintf("transient error while getting job %q status: %s", id, err),
					job:     job,
				})
			}
			return options.error(&BackupJobCompletionError{
				message: fmt.Sprintf("an error occured while getting job %q status: %s", id, err),
				job:     job,
			})
		}
		if job == nil {
			err := &BackupJobCompletionError{
				message: fmt.Sprintf("the job %q could not be found", id),
			}
			if count == 1 {
				return options.retryableError(err)
			}
			return options.error(err)
		}
		res = job
		switch job.Status {
		case "IDLE":
			options.log(fmt.Sprintf("the job %q is completed", id))
			return nil
		case "RUNNING":
			return options.retryableError(&BackupJobCompletionError{
				message: fmt.Sprintf("the job %q is still running", id),
				job:     job,
			})
		default:
			return options.error(&BackupJobCompletionError{
				message: fmt.Sprintf("the job %q has failed", id),
				job:     job,
			})
		}
	})

	return res, err
}
