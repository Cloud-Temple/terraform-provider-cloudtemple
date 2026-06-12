package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// maxTransientVIFAttempts bounds the TOTAL number of attempts (not retries)
// of a VIF PATCH whose activity failed for a transient platform reason
// ("None of the workers were able to respond", #251).
const maxTransientVIFAttempts = 3

// vifUpdateFuncs abstracts the API surface of the bounded VIF retry so the
// loop is unit tested without HTTP calls or sleeps.
type vifUpdateFuncs struct {
	// read returns the live adapter (nil when it no longer exists).
	read func(ctx context.Context) (*client.OpenIaaSNetworkAdapter, error)
	// update starts the PATCH and returns the activity id.
	update func(ctx context.Context, req *client.UpdateOpenIaasNetworkAdapterRequest) (string, error)
	// wait waits for the activity completion.
	wait func(ctx context.Context, activityID string) (*client.Activity, error)
	// sleep waits between attempts; it returns ctx.Err() on cancellation.
	sleep func(ctx context.Context, attempt int) error
	// isTransient classifies a completion failure as retryable; defaults
	// to client.IsTransientActivityFailure (injectable for tests, whose
	// package cannot build the unexported ActivityCompletionError fields).
	isTransient func(err error) bool
}

func defaultVIFSleep(ctx context.Context, attempt int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(attempt) * 10 * time.Second):
		return nil
	}
}

// runVIFUpdateWithRetry applies a VIF PATCH with a bounded retry on
// transient platform failures. Invariants defended by the unit tests
// (#264 plan, pre-written before this implementation):
//   - the payload is rebuilt against the freshly read live adapter BEFORE
//     every attempt: if a previous attempt actually converged server-side,
//     re-sending the stale payload would be rejected as a VPC static-IP
//     self-conflict — a nil rebuilt payload means converged, success;
//   - only the narrowly matched transient platform reason is retried
//     (client.IsTransientActivityFailure); any other failure is immediate;
//   - at most maxTransientVIFAttempts TOTAL attempts;
//   - no extra call after a success;
//   - an adapter that disappeared (read returns nil) is an explicit error,
//     never an opportunity to act on a stale id.
func runVIFUpdateWithRetry(ctx context.Context, adapterID string, funcs vifUpdateFuncs, build func(actual *client.OpenIaaSNetworkAdapter) *client.UpdateOpenIaasNetworkAdapterRequest) error {
	if funcs.sleep == nil {
		funcs.sleep = defaultVIFSleep
	}
	if funcs.isTransient == nil {
		funcs.isTransient = client.IsTransientActivityFailure
	}

	var lastErr error
	for attempt := 1; attempt <= maxTransientVIFAttempts; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			if lastErr != nil {
				return fmt.Errorf("cancelled while retrying: %w (last transient failure: %s)", ctxErr, lastErr)
			}
			return ctxErr
		}
		actual, err := funcs.read(ctx)
		if err != nil {
			return fmt.Errorf("failed to read network adapter %s: %w", adapterID, err)
		}
		if actual == nil {
			return fmt.Errorf("network adapter %s not found", adapterID)
		}

		req := build(actual)
		if req == nil {
			// The adapter is already converged: nothing to push. This is
			// the normal exit when a transiently-failed attempt actually
			// succeeded server-side.
			return nil
		}

		activityID, err := funcs.update(ctx, req)
		if err != nil {
			return err
		}
		_, err = funcs.wait(ctx, activityID)
		if err == nil {
			return nil
		}
		if !funcs.isTransient(err) {
			return err
		}
		lastErr = err

		if attempt == maxTransientVIFAttempts {
			break
		}
		tflog.Warn(ctx, fmt.Sprintf("update network adapter %s: transient platform failure (attempt %d/%d), retrying: %s",
			adapterID, attempt, maxTransientVIFAttempts, err))
		if sleepErr := funcs.sleep(ctx, attempt); sleepErr != nil {
			return fmt.Errorf("cancelled while retrying: %w (last transient failure: %s)", sleepErr, lastErr)
		}
	}
	return lastErr
}

// clientVIFUpdateFuncs wires the retry loop to the real API client.
func clientVIFUpdateFuncs(c *client.Client, adapterID string, options *client.WaiterOptions) vifUpdateFuncs {
	return vifUpdateFuncs{
		read: func(ctx context.Context) (*client.OpenIaaSNetworkAdapter, error) {
			return c.Compute().OpenIaaS().NetworkAdapter().Read(ctx, adapterID)
		},
		update: func(ctx context.Context, req *client.UpdateOpenIaasNetworkAdapterRequest) (string, error) {
			return c.Compute().OpenIaaS().NetworkAdapter().Update(ctx, adapterID, req)
		},
		wait: func(ctx context.Context, activityID string) (*client.Activity, error) {
			return c.Activity().WaitForCompletion(ctx, activityID, options)
		},
	}
}
