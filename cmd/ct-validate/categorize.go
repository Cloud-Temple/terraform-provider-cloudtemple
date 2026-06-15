package main

import (
	"context"
	"errors"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// Category classifies the outcome of an operation. It is the histogram key in
// the report and the input to the circuit breaker's failure accounting.
type Category string

const (
	// CategoryOK is a successful operation.
	CategoryOK Category = "ok"
	// CategorySkipped is an op deliberately not attempted (e.g. no spare FIP).
	// It is neither a success nor a failure.
	CategorySkipped Category = "skipped"
	// CategoryTransientWorkers is the known-transient platform reason
	// "None of the workers were able to respond" (VPC workers, #251).
	CategoryTransientWorkers Category = "transient_workers"
	// CategoryBadGateway502 is a 502 / gateway / config-load failure, the
	// signature of the 2026-06-15 sustained API outage.
	CategoryBadGateway502 Category = "bad_gateway_502"
	// CategoryHTTP4xx is a client-side HTTP status (400-499) other than ones
	// captured by a more specific category.
	CategoryHTTP4xx Category = "http_4xx"
	// CategoryHTTP5xx is a server-side HTTP status (500-599) other than 502.
	CategoryHTTP5xx Category = "http_5xx"
	// CategoryTimeout is a context deadline or cancellation.
	CategoryTimeout Category = "timeout"
	// CategoryOther is anything not matched above (decode errors, transport
	// errors without a status, etc.).
	CategoryOther Category = "other"
)

// categorize maps an operation error to a Category. A nil error is CategoryOK.
//
// Ordering matters and is deliberate:
//  1. nil               → OK
//  2. context dl/cancel → Timeout (a cancelled write may still surface a
//     wrapped status; the cancellation is the root cause and must win).
//  3. transient workers → TransientWorkers (platform-recoverable).
//  4. 502 / bad gateway → BadGateway502 (the outage signature; checked before
//     the generic 5xx bucket so it is never swallowed by it).
//  5. StatusError 4xx   → HTTP4xx
//  6. StatusError 5xx   → HTTP5xx
//  7. fallback          → Other
//
// Every branch is exercised by a mutation-proven unit test.
func categorize(err error) Category {
	if err == nil {
		return CategoryOK
	}

	// Timeout / cancellation takes precedence: it is the operator-facing root
	// cause when the global -timeout fires or the breaker cancels work.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return CategoryTimeout
	}

	// Platform-recoverable "workers unavailable" reason, recognised either via
	// the typed client helper or via the raw message (some error paths wrap it
	// as a plain string rather than an ActivityCompletionError).
	if client.IsTransientActivityFailure(err) || containsAny(err.Error(), "None of the workers were able to respond") {
		return CategoryTransientWorkers
	}

	msg := err.Error()
	if containsAny(msg, "502", "Bad Gateway", "Failed to load configuration via API") {
		return CategoryBadGateway502
	}

	var statusErr client.StatusError
	if errors.As(err, &statusErr) {
		switch {
		case statusErr.Code >= 400 && statusErr.Code <= 499:
			return CategoryHTTP4xx
		case statusErr.Code >= 500 && statusErr.Code <= 599:
			return CategoryHTTP5xx
		}
	}

	return CategoryOther
}

// containsAny reports whether s contains at least one of the needles.
func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// isFailure reports whether a category counts as a failure for the circuit
// breaker. OK and Skipped do not; everything else does.
func (c Category) isFailure() bool {
	return c != CategoryOK && c != CategorySkipped
}
