package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sethvargo/go-retry"
)

// transientActivityReasons lists platform-side failure reasons known to be
// temporary (VPC workers not responding) and safe to retry at the operation
// level (#251). Permanent reasons (MAC conflict, insufficient space…) must
// stay immediately fatal.
var transientActivityReasons = []string{
	"None of the workers were able to respond",
}

// IsTransientActivityFailure reports whether err is an activity that reached
// the "failed" state for a reason known to be transient platform-side.
func IsTransientActivityFailure(err error) bool {
	var ace *ActivityCompletionError
	if !errors.As(err, &ace) || ace.activity == nil {
		return false
	}
	for _, state := range ace.activity.State {
		for _, marker := range transientActivityReasons {
			if strings.Contains(state.Reason, marker) {
				return true
			}
		}
	}
	return false
}

type ActivityClient struct {
	c *Client
}

func (c *Client) Activity() *ActivityClient {
	return &ActivityClient{c}
}

type Activity struct {
	ID             string
	TenantId       string
	Description    string
	Type           string
	Tags           []string
	CreationDate   time.Time
	ConcernedItems []ActivityConcernedItem
	State          map[string]ActivityState
}

type ActivityState struct {
	StartDate   string
	StopDate    string
	Reason      string
	Result      string
	Progression float64
}

type ActivityConcernedItem struct {
	ID   string
	Type string
}

func (c *ActivityClient) List(ctx context.Context, filter *struct{}) ([]*Activity, error) {
	r := c.c.newRequest("GET", "/activity/v1/activities")
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
	r := c.c.newRequest("GET", "/activity/v1/activities/%s", id)
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

// maxActivityReadRetries bounds the number of CONSECUTIVE transient read
// failures (5xx, throttling, transport errors) tolerated while polling an
// activity. Without it, a single transient 500 while reading the activity
// status fails an operation that keeps running platform-side, leaving the
// resource orphaned outside the Terraform state (issue #245).
const maxActivityReadRetries = 8

func (c *ActivityClient) WaitForCompletion(ctx context.Context, id string, options *WaiterOptions) (*Activity, error) {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithCappedDuration(30*time.Second, b)

	return waitForActivityCompletion(ctx, id, func(ctx context.Context) (*Activity, error) {
		return c.Read(ctx, id)
	}, b, options)
}

// activityReadFunc abstracts the activity read so the polling loop can be
// unit tested without HTTP calls or sleeps.
type activityReadFunc func(ctx context.Context) (*Activity, error)

// waitForActivityCompletion is the polling loop behind WaitForCompletion,
// with the read and the backoff injected. Invariants defended by the unit
// tests (#245, #264 plan):
//   - transient read failures (429/5xx/transport) are retried with a
//     bounded CONSECUTIVE budget, reset by any successful read;
//   - permanent read errors (4xx, decode, context cancellation) fail
//     immediately;
//   - a terminal "failed" activity is never retried;
//   - an initial not-found is tolerated once (eventual consistency),
//     a repeated not-found is permanent.
func waitForActivityCompletion(ctx context.Context, id string, read activityReadFunc, b retry.Backoff, options *WaiterOptions) (*Activity, error) {
	var res *Activity
	var count int
	var consecutiveReadFailures int

	err := retry.Do(ctx, b, func(ctx context.Context) error {
		count++
		activity, err := read(ctx)
		if err != nil {
			if isTransientAPIError(err) && consecutiveReadFailures < maxActivityReadRetries {
				consecutiveReadFailures++
				return options.retryableError(&ActivityCompletionError{
					message: fmt.Sprintf("transient error while getting the status of activity %q (attempt %d/%d): %s",
						id, consecutiveReadFailures, maxActivityReadRetries, err),
				})
			}
			return options.error(&ActivityCompletionError{
				message: fmt.Sprintf("an error occured while getting the status of activity %q: %s", id, err),
			})
		}
		consecutiveReadFailures = 0

		if activity == nil {
			err := &ActivityCompletionError{
				message: fmt.Sprintf("the activity %q could not be found", id),
			}
			if count == 1 {
				return options.retryableError(err)
			}
			return options.error(err)
		}
		if len(activity.State) != 1 {
			return options.retryableError(&ActivityCompletionError{
				message: fmt.Sprintf("unexpected state for activity %q: %v", id, activity.State),
			})
		}
		res = activity
		for state := range activity.State {
			switch state {
			case "completed":
				options.log(fmt.Sprintf("the activity %q is completed", id))
				return nil
			case "failed":
				return options.error(&ActivityCompletionError{
					activity: activity,
				})
			default:
				return options.retryableError(&ActivityCompletionError{
					message:  fmt.Sprintf("unexpected state for activity %q: %v", id, state),
					activity: activity,
				})
			}
		}

		options.log(fmt.Sprintf("no state found for activity %q", id))
		return nil
	})

	return res, err
}
