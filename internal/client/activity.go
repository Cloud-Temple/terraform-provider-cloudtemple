package client

import (
	"context"
	"fmt"
	"time"

	"github.com/sethvargo/go-retry"
)

type ActivityClient struct {
	c *Client
}

func (c *Client) Activity() *ActivityClient {
	return &ActivityClient{c}
}

type Activity struct {
	ID             string                   `terraform:"id"`
	TenantId       string                   `terraform:"tenant_id"`
	Description    string                   `terraform:"description"`
	Type           string                   `terraform:"type"`
	Tags           []string                 `terraform:"tags"`
	CreationDate   time.Time                `terraform:"creation_date"`
	ConcernedItems []ActivityConcernedItem  `terraform:"concerned_items"`
	State          map[string]ActivityState `terraform:"-"`
}

type ActivityState struct {
	StartDate   string
	StopDate    string
	Reason      string
	Progression int
}

type ActivityConcernedItem struct {
	ID   string `terraform:"id"`
	Type string `terraform:"type"`
}

func (c *ActivityClient) List(ctx context.Context, filter *struct{}) ([]*Activity, error) {
	r := c.c.newRequest("GET", "/api/activity/v1/activities")
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	if err := requireOK(resp); err != nil {
		return nil, err
	}

	var out []*Activity
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (c *ActivityClient) Read(ctx context.Context, id string) (*Activity, error) {
	r := c.c.newRequest("GET", "/api/activity/v1/activities/"+id)
	resp, err := c.c.doRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	defer closeResponseBody(resp)
	found, err := requireNotFoundOrOK(resp, 404)
	if err != nil || !found {
		return nil, err
	}

	var out Activity
	if err := decodeBody(resp, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *ActivityClient) WaitForCompletion(ctx context.Context, id string) (*Activity, error) {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithCappedDuration(30*time.Second, b)

	var res *Activity

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		activity, err := c.Read(ctx, id)
		if err != nil {
			return retry.RetryableError(fmt.Errorf("an error occured while getting activity status: %v", err))
		}
		if len(activity.State) != 1 {
			return retry.RetryableError(fmt.Errorf("unexpected state: %v", activity.State))
		}
		res = activity
		for state := range activity.State {
			switch state {
			case "completed":
				return nil
			case "failed":
				return fmt.Errorf("the activity has failed: %v", activity.State["failed"].Reason)
			default:
				return retry.RetryableError(fmt.Errorf("unexpected state: %v", state))
			}
		}
		return nil

	})

	return res, err
}
