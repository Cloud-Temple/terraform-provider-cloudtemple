package client

import (
	"context"
	"fmt"
	"strings"
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
	Result      string
	Progression float64
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
	r := c.c.newRequest("GET", "/api/activity/v1/activities/%s", id)
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

type ActivityCompletionError struct {
	message  string
	activity *Activity
}

const activityErrorMessage = `

  Description: %s
  Tenant ID: %q
  Created at %s
  Type: %s
  Tags: %s

  Concerned Items:
%s

%s`

func (a *ActivityCompletionError) Error() string {
	message := a.message
	if message == "" {
		message = fmt.Sprintf("an error occured while waiting for completion of activity %q:", a.activity.ID)
	}

	if a.activity != nil {
		var concernedItemMessage []string
		for _, concernedItem := range a.activity.ConcernedItems {
			concernedItemMessage = append(
				concernedItemMessage,
				fmt.Sprintf("    - ID: %q\n      Type: %s", concernedItem.ID, concernedItem.Type),
			)
		}
		if len(concernedItemMessage) == 0 {
			concernedItemMessage = []string{"    none"}
		}

		var stateMessage []string
		for name, state := range a.activity.State {
			stateMessage = append(
				stateMessage,
				fmt.Sprintf(
					"  State: %s\n    Result: %s\n    Reason: %s\n    Started at %s\n    Stopped at %s",
					name,
					state.Result,
					state.Reason,
					state.StartDate,
					state.StopDate,
				),
			)
		}

		message += fmt.Sprintf(
			activityErrorMessage,
			a.activity.Description,
			a.activity.TenantId,
			a.activity.CreationDate.String(),
			a.activity.Type,
			strings.Join(a.activity.Tags, ", "),
			strings.Join(concernedItemMessage, "\n"),
			strings.Join(stateMessage, "\n"),
		)
	}

	return message
}

func (c *ActivityClient) WaitForCompletion(ctx context.Context, id string) (*Activity, error) {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithCappedDuration(30*time.Second, b)

	var res *Activity
	var count int

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		count++
		activity, err := c.Read(ctx, id)
		if err != nil {
			return retry.RetryableError(&ActivityCompletionError{
				message:  fmt.Sprintf("an error occured while getting the status of activity %q: %s", id, err),
				activity: activity,
			})
		}
		if activity == nil {
			err := &ActivityCompletionError{
				message: fmt.Sprintf("the activity %q could not be found", id),
			}
			if count == 1 {
				return retry.RetryableError(err)
			}
			return err
		}
		if len(activity.State) != 1 {
			return retry.RetryableError(&ActivityCompletionError{
				message: fmt.Sprintf("unexpected state for activity %q: %v", id, activity.State),
			})
		}
		res = activity
		for state := range activity.State {
			switch state {
			case "completed":
				return nil
			case "failed":
				return &ActivityCompletionError{
					activity: activity,
				}
			default:
				return retry.RetryableError(&ActivityCompletionError{
					message:  fmt.Sprintf("unexpected state for activity %q: %v", id, state),
					activity: activity,
				})
			}
		}
		return nil

	})

	return res, err
}
