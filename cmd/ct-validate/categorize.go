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
	// CategoryRateLimited is HTTP 429 Too Many Requests. It is a 4xx by status
	// but, unlike the other client errors, it IS a distress signal ("back off"),
	// so it is split out and DOES trip the breaker (see isDistress).
	CategoryRateLimited Category = "rate_limited"
	// CategoryHTTP4xx is a client-side HTTP status (400-499) other than ones
	// captured by a more specific category (e.g. 429). These are deterministic
	// client errors (bad request, not authorized, forbidden, not found,
	// conflict) — a real failure to report, but NOT API distress.
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
//  5. StatusError 429   → RateLimited (a 4xx, but a "back off" distress signal)
//  6. StatusError 4xx   → HTTP4xx
//  7. StatusError 5xx   → HTTP5xx
//  8. fallback          → Other
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
		case statusErr.Code == 429:
			return CategoryRateLimited
		case statusErr.Code >= 400 && statusErr.Code <= 499:
			return CategoryHTTP4xx
		case statusErr.Code >= 500 && statusErr.Code <= 599:
			return CategoryHTTP5xx
		}
	}

	return CategoryOther
}

// isDistress reports whether a category indicates the shared API is in DISTRESS,
// so the circuit breaker should trip and stop launching work. It is deliberately
// NARROWER than "is this a failure": a deterministic client error (HTTP 4xx —
// tenant not entitled to a service, missing filter, 404, conflict) is a real
// failure to REPORT, but it is NOT distress and must not trip the breaker and
// mask the rest of the map. Rate limiting (429) and server/transport distress DO
// trip. OK and Skipped are never distress.
//
// CategoryOther (transport error without a status, or an unexpected
// decode/contract error) is treated as distress on purpose: fail-safe, an
// UNCLASSIFIED error against a shared API is a reason to back off. A deterministic
// decode error could in theory re-trip like the old 4xx did; if that ever shows
// up in practice, split Other into transport vs decode and only trip on transport.
func (c Category) isDistress() bool {
	switch c {
	case CategoryTransientWorkers, CategoryBadGateway502, CategoryHTTP5xx,
		CategoryTimeout, CategoryRateLimited, CategoryOther:
		return true
	default: // CategoryOK, CategorySkipped, CategoryHTTP4xx
		return false
	}
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
