package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

func TestCategorize(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Category
	}{
		{"nil is OK", nil, CategoryOK},
		{"deadline is Timeout", context.DeadlineExceeded, CategoryTimeout},
		{"cancel is Timeout", context.Canceled, CategoryTimeout},
		{
			"wrapped deadline is Timeout",
			fmt.Errorf("doing thing: %w", context.DeadlineExceeded),
			CategoryTimeout,
		},
		{
			"workers message is TransientWorkers",
			errors.New("activity failed: None of the workers were able to respond"),
			CategoryTransientWorkers,
		},
		{
			"502 is BadGateway502",
			client.StatusError{Code: 502, Body: "Bad Gateway"},
			CategoryBadGateway502,
		},
		{
			"bad gateway text is BadGateway502",
			errors.New("upstream: Bad Gateway"),
			CategoryBadGateway502,
		},
		{
			"load configuration message is BadGateway502",
			errors.New("Failed to load configuration via API"),
			CategoryBadGateway502,
		},
		{
			"400 is HTTP4xx",
			client.StatusError{Code: 400, Body: "bad request"},
			CategoryHTTP4xx,
		},
		{
			"404 is HTTP4xx",
			client.StatusError{Code: 404, Body: "not found"},
			CategoryHTTP4xx,
		},
		{
			"500 is HTTP5xx",
			client.StatusError{Code: 500, Body: "boom"},
			CategoryHTTP5xx,
		},
		{
			"503 is HTTP5xx",
			client.StatusError{Code: 503, Body: "unavailable"},
			CategoryHTTP5xx,
		},
		{
			"plain error is Other",
			errors.New("json: cannot unmarshal"),
			CategoryOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := categorize(tt.err); got != tt.want {
				t.Fatalf("categorize(%v) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

// TestCategorizeTimeoutBeatsStatus pins the precedence: a cancellation that
// also carries a status must categorize as Timeout, not as the status. This is
// the branch a "reorder the checks" mutation would break.
func TestCategorizeTimeoutBeatsStatus(t *testing.T) {
	// A 503 StatusError wrapped together with a cancellation: the cancellation
	// is the operator-facing root cause.
	err := fmt.Errorf("call aborted (%w): %w", context.Canceled, client.StatusError{Code: 503})
	if got := categorize(err); got != CategoryTimeout {
		t.Fatalf("expected Timeout to win over 5xx, got %q", got)
	}
}

// TestCategorize502BeatsGeneric5xx pins that 502 is classified as BadGateway502
// and NOT swallowed by the generic 5xx bucket. A mutation that drops the 502
// pre-check (letting the StatusError switch handle it) would turn this RED.
func TestCategorize502BeatsGeneric5xx(t *testing.T) {
	if got := categorize(client.StatusError{Code: 502, Body: "x"}); got != CategoryBadGateway502 {
		t.Fatalf("502 must be BadGateway502, got %q", got)
	}
}

func TestCategoryIsFailure(t *testing.T) {
	failures := []Category{
		CategoryTransientWorkers, CategoryBadGateway502, CategoryHTTP4xx,
		CategoryHTTP5xx, CategoryTimeout, CategoryOther,
	}
	for _, c := range failures {
		if !c.isFailure() {
			t.Errorf("%q should count as failure", c)
		}
	}
	for _, c := range []Category{CategoryOK, CategorySkipped} {
		if c.isFailure() {
			t.Errorf("%q must NOT count as failure", c)
		}
	}
}
