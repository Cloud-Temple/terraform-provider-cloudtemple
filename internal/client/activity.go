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

// transientActivityReasonPairs lists COMPOSITE (AND) markers: a failure reason
// is transient only when it contains the lead phrase AND at least one of the
// corroborating signals. The VPC platform's transient gateway hiccup
// (#315/#319) surfaces as "Failed to load configuration via API: <html>…502
// Bad Gateway…nginx…". The lead phrase ALONE is too broad: IsTransientActivity
// Failure is GLOBAL — it also gates VIF/compute retries — and a genuine
// PERMANENT config-load failure could carry the same phrase. Requiring a
// 502 / Bad Gateway corroboration keeps it fail-closed: a false negative
// (missing a transient 502) is preferred to a false positive (retrying a
// non-idempotent permanent failure). A bare "502" without the lead phrase is
// likewise NOT matched, so an unrelated upstream 502 stays fatal.
var transientActivityReasonPairs = []struct {
	lead  string
	anyOf []string
}{
	{
		lead:  "Failed to load configuration via API",
		anyOf: []string{"502", "Bad Gateway"},
	},
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
		for _, pair := range transientActivityReasonPairs {
			if strings.Contains(state.Reason, pair.lead) && containsAny(state.Reason, pair.anyOf) {
				return true
			}
		}
	}
	return false
}

// containsAny reports whether s contains at least one of the given substrings.
func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
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
//   - initial not-founds are tolerated up to a bounded budget
//     (WaiterOptions.NotFoundRetries, default 1) for eventual consistency,
//     a not-found beyond the budget is permanent;
//   - an activity that was seen and then vanishes is always permanent,
//     regardless of the not-found budget.
func waitForActivityCompletion(ctx context.Context, id string, read activityReadFunc, b retry.Backoff, options *WaiterOptions) (*Activity, error) {
	var res *Activity
	var consecutiveReadFailures int
	// The initial not-found tolerance (eventual consistency right after the
	// activity is started) is tracked independently from the transient read
	// budget: a 429/5xx/transport blip before the first successful read must
	// not consume it (FF-3). Its size is WaiterOptions.NotFoundRetries (default
	// 1) so a slow-to-index activity does not fail a write that keeps running
	// platform-side (#415). The disappearance of an activity that was already
	// seen stays permanent.
	var notFoundReads int
	notFoundBudget := options.notFoundBudget()
	var activitySeen bool

	err := retry.Do(ctx, b, func(ctx context.Context) error {
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
			if activitySeen {
				// An activity that was visible and vanished is permanent,
				// regardless of the not-found budget.
				return options.error(err)
			}
			if notFoundReads < notFoundBudget {
				notFoundReads++
				return options.retryableError(err)
			}
			return options.error(err)
		}
		activitySeen = true
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
