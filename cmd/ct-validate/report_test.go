package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSqueaksOrderedWorstFirst(t *testing.T) {
	stats := []EndpointStats{
		{Cycle: "readonly", Endpoint: "good", Total: 4, OK: 4},       // 100% — not a squeak
		{Cycle: "readonly", Endpoint: "mid", Total: 4, OK: 2},        // 50%
		{Cycle: "readonly", Endpoint: "bad", Total: 4, OK: 0},        // 0%
		{Cycle: "readonly", Endpoint: "skips", Total: 2, Skipped: 2}, // 100% (skips) — not a squeak
	}
	sq := squeaks(stats)
	if len(sq) != 2 {
		t.Fatalf("want 2 squeaks (mid,bad), got %d: %+v", len(sq), sq)
	}
	// Worst (0%) must come first.
	if sq[0].Endpoint != "bad" || sq[1].Endpoint != "mid" {
		t.Fatalf("squeaks not worst-first: %+v", sq)
	}
}

func TestTopReasonsExcludesOKAndSkipped(t *testing.T) {
	reasons := map[Category]int{
		CategoryOK:               10,
		CategorySkipped:          3,
		CategoryHTTP5xx:          5,
		CategoryBadGateway502:    2,
		CategoryTransientWorkers: 1,
	}
	got := topReasons(reasons, 2)
	if strings.Contains(got, "ok") || strings.Contains(got, "skipped") {
		t.Fatalf("topReasons leaked ok/skipped: %q", got)
	}
	// Most frequent first, capped at 2.
	if !strings.HasPrefix(got, "http_5xx=5") {
		t.Fatalf("topReasons should lead with the most frequent failure: %q", got)
	}
	if strings.Count(got, "=") != 2 {
		t.Fatalf("topReasons should cap at 2 entries: %q", got)
	}
}

func TestTopReasonsNoFailures(t *testing.T) {
	if got := topReasons(map[Category]int{CategoryOK: 5}, 3); got != "-" {
		t.Fatalf("no failures should render as '-', got %q", got)
	}
}

func TestPrintTextSurfacesTripAndSqueaks(t *testing.T) {
	res := EngineResult{
		SelectedCycles: []string{"readonly"},
		Tripped:        true,
		TripReason:     "5 consecutive failures (limit 5)",
		Stats: []EndpointStats{
			{Cycle: "readonly", Endpoint: "vpc.vpc.list", Total: 5, OK: 1, Reasons: map[Category]int{CategoryBadGateway502: 4, CategoryOK: 1}},
		},
	}
	var buf bytes.Buffer
	PrintText(&buf, res)
	out := buf.String()
	for _, want := range []string{"CIRCUIT BREAKER TRIPPED", "WHERE IT SQUEAKS", "vpc.vpc.list", "FAILURES PRESENT"} {
		if !strings.Contains(out, want) {
			t.Fatalf("text report missing %q\n---\n%s", want, out)
		}
	}
}

// TestAggregateCapturesFirstFailureDetail pins that the per-endpoint SampleError
// is the FIRST failed op's Detail (the reason it squeaked), and that OK/skipped
// ops never set or overwrite it. Mutation: drop the SampleError capture in
// Aggregate → SampleError stays empty → RED.
func TestAggregateCapturesFirstFailureDetail(t *testing.T) {
	ops := []Op{
		{Cycle: "c", Endpoint: "e", OK: true, Category: CategoryOK},
		{Cycle: "c", Endpoint: "e", OK: false, Category: CategoryHTTP4xx, Detail: "Unexpected response code: 403 (Forbidden.)"},
		{Cycle: "c", Endpoint: "e", OK: false, Category: CategoryHTTP4xx, Detail: "second failure, must NOT overwrite"},
		{Cycle: "c", Endpoint: "e", Skipped: true, Category: CategorySkipped, Detail: "skips carry none"},
	}
	stats := Aggregate(ops)
	if len(stats) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(stats))
	}
	if got := stats[0].SampleError; got != "Unexpected response code: 403 (Forbidden.)" {
		t.Fatalf("SampleError must be the FIRST failure's detail, got %q", got)
	}
}

// TestPrintTextShowsFailureDetail pins that a SampleError is rendered under WHERE
// IT SQUEAKS so the operator sees WHY, not just the category. Mutation: drop the
// SampleError render in PrintText → the body line disappears → RED.
func TestPrintTextShowsFailureDetail(t *testing.T) {
	res := EngineResult{
		SelectedCycles: []string{"compute_vmware_lifecycle"},
		Stats: []EndpointStats{
			{Cycle: "compute_vmware_lifecycle", Endpoint: "compute.vmware.virtual_machine.create",
				Total: 1, OK: 0, Reasons: map[Category]int{CategoryHTTP4xx: 1},
				SampleError: "Unexpected response code: 403 (Forbidden.)"},
		},
	}
	var buf bytes.Buffer
	PrintText(&buf, res)
	if !strings.Contains(buf.String(), "403 (Forbidden.)") {
		t.Fatalf("the failure body must be surfaced under WHERE IT SQUEAKS\n---\n%s", buf.String())
	}
}

// TestTruncateCollapsesAndBounds pins truncate: it collapses newlines to keep one
// readable line and bounds the length with an ellipsis.
func TestTruncateCollapsesAndBounds(t *testing.T) {
	if got := truncate("a\nb\rc", 100); got != "a b c" {
		t.Fatalf("newlines must collapse to spaces, got %q", got)
	}
	long := strings.Repeat("x", 500)
	got := truncate(long, 300)
	if len([]rune(got)) != 301 || !strings.HasSuffix(got, "…") {
		t.Fatalf("must bound to 300 runes + ellipsis, got len=%d suffix=%v", len([]rune(got)), strings.HasSuffix(got, "…"))
	}
	if got := truncate("short", 300); got != "short" {
		t.Fatalf("a short string must be unchanged, got %q", got)
	}
	// n<=0 must be safe (no panic on r[:n]) and yield empty.
	if got := truncate("x", 0); got != "" {
		t.Fatalf("truncate(_,0) must be empty, got %q", got)
	}
	if got := truncate("x", -5); got != "" {
		t.Fatalf("truncate(_,negative) must be empty (no panic), got %q", got)
	}
}

// TestRedactSecretsMasksCredentials pins that recorded error text never surfaces a
// credential: bearer tokens and password/secret/token/authorization carriers are
// masked, while an ordinary error body is left intact. Mutation: make
// redactSecrets a pass-through → the token leaks → RED.
func TestRedactSecretsMasksCredentials(t *testing.T) {
	cases := map[string]string{
		"Authorization: Bearer eyJhbGciOiJ.payload.sig":              "eyJhbGciOiJ",
		"Authorization: Basic dXNlcjpwYXNzd29yZGxvbmdlbm91Z2g":       "dXNlcjpwYXNzd29yZGxvbmdlbm91Z2g",
		"Authorization: Token 0a1b2c3d4e5f6071829304a5b6c7":          "0a1b2c3d4e5f6071829304a5b6c7",
		"Authorization: ApiKey deadbeefcafedeadbeefcafe":             "deadbeefcafedeadbeefcafe",
		"{\"authorization\":\"Basic c2VjcmV0dmFsdWVsb25nZW5vdWdo\"}": "c2VjcmV0dmFsdWVsb25nZW5vdWdo",
		"client_secret=8f5767aa-dead-beef&grant=x":                   "8f5767aa-dead-beef",
		"access_token: ya29.A0ARrdaMlongtokenvalue":                  "ya29.A0ARrdaMlongtokenvalue",
		"refresh_token=1//0longrefreshtokenvalue":                    "0longrefreshtokenvalue",
		"signature=\"aGVsbG8gd29ybGQgc2lnbmF0dXJl\"":                 "aGVsbG8gd29ybGQgc2lnbmF0dXJl",
		"\"token\": \"abcdef123456\"":                                "abcdef123456",
		"password = hunter2":                                         "hunter2",
		"password=\"correct horse battery staple\"":                  "battery",
		"bearer eyJraw.token.here.long.enough.value":                 "eyJraw.token.here.long.enough.value",
		"leaked raw blob 0123456789abcdef0123456789abcdef":           "0123456789abcdef0123456789abcdef",
		// Digitless secrets must STILL be masked — the reason-code exception is narrow.
		"Bearer AAAAAAAAAAAAAAAAAAAAAAAA":   "AAAAAAAAAAAAAAAAAAAAAAAA",
		"token=ABCDEFGHIJKLMNOPQRSTUVWX":    "ABCDEFGHIJKLMNOPQRSTUVWX",
		"opaque abcdefghijklmnopqrstuvwxyz": "abcdefghijklmnopqrstuvwxyz",
	}
	for in, leak := range cases {
		out := redactSecrets(in)
		if strings.Contains(out, leak) {
			t.Fatalf("redactSecrets leaked %q: in=%q out=%q", leak, in, out)
		}
		if !strings.Contains(out, "REDACTED") {
			t.Fatalf("redactSecrets must mark a redaction: in=%q out=%q", in, out)
		}
	}
	// A non-secret key after a secret query param must survive (no over-greedy match).
	if out := redactSecrets("client_secret=abc&foo=bar"); !strings.Contains(out, "foo=bar") {
		t.Fatalf("a benign trailing param must survive redaction, got %q", out)
	}
	// An ordinary API error body must be left intact (no false redaction).
	if got := redactSecrets("Unexpected response code: 403 (Forbidden.)"); got != "Unexpected response code: 403 (Forbidden.)" {
		t.Fatalf("an ordinary body must be unchanged, got %q", got)
	}
}

// TestRedactSecretsPreservesErrorReasonCodes pins that the catch-all opaque-token
// mask does NOT swallow long all-letter/underscore error reason codes (no digit) —
// those are the diagnostic the report exists to surface (a real run failed with
// MEMORY_CONSTRAINT_VIOLATION_ORDER and the mask was hiding it). A token-like run
// WITH a digit is still masked. Mutation: drop the digit requirement → the reason
// code is masked → RED.
func TestRedactSecretsPreservesErrorReasonCodes(t *testing.T) {
	for _, keep := range []string{
		"activity failed: MEMORY_CONSTRAINT_VIOLATION_ORDER",
		"VM state is halted but should be paused, suspended, running",
		"the virtual machine could not be created: INSUFFICIENT_RESOURCES_AVAILABLE",
		"order rejected: ERROR_500_INTERNAL_FAILURE", // reason code WITH a digit segment
	} {
		if got := redactSecrets(keep); got != keep {
			t.Fatalf("an all-letter error reason code must be preserved, got %q from %q", got, keep)
		}
	}
	// A digit-bearing opaque blob (realistic credential material) is STILL masked.
	if got := redactSecrets("dump 0123456789abcdef0123456789abcdef"); strings.Contains(got, "0123456789abcdef0123456789abcdef") {
		t.Fatalf("a digit-bearing opaque blob must still be masked, got %q", got)
	}
}

// TestAggregateSkipBeforeFailureDoesNotSetSample pins that a skipped op preceding
// a real failure does not become the sample (only a real failure's detail does).
func TestAggregateSkipBeforeFailureDoesNotSetSample(t *testing.T) {
	ops := []Op{
		{Cycle: "c", Endpoint: "e", Skipped: true, Category: CategorySkipped, Detail: "must be ignored"},
		{Cycle: "c", Endpoint: "e", OK: false, Category: CategoryHTTP5xx, Detail: "the real reason"},
	}
	stats := Aggregate(ops)
	if stats[0].SampleError != "the real reason" {
		t.Fatalf("a skip before the failure must not set the sample, got %q", stats[0].SampleError)
	}
}

func TestPrintJSONShape(t *testing.T) {
	res := EngineResult{
		SelectedCycles:  []string{"readonly"},
		GatedWriteSkips: []string{"vpc"},
		Stats: []EndpointStats{
			{Cycle: "readonly", Endpoint: "iam.users.list", Total: 2, OK: 1, Reasons: map[Category]int{CategoryHTTP5xx: 1, CategoryOK: 1}, SampleError: "boom 500"},
		},
	}
	var buf bytes.Buffer
	if err := PrintJSON(&buf, res); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	var rep jsonReport
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("JSON not parseable: %v\n%s", err, buf.String())
	}
	if !rep.HasFailure {
		t.Fatal("JSON has_failure should be true (one endpoint at 50%)")
	}
	if len(rep.Endpoints) != 1 || rep.Endpoints[0].SuccessRate != 0.5 {
		t.Fatalf("JSON endpoint shape wrong: %+v", rep.Endpoints)
	}
	if rep.Endpoints[0].SampleError != "boom 500" {
		t.Fatalf("JSON must carry sample_error, got %q", rep.Endpoints[0].SampleError)
	}
	if len(rep.GatedWrite) != 1 || rep.GatedWrite[0] != "vpc" {
		t.Fatalf("JSON gated_write_cycles wrong: %+v", rep.GatedWrite)
	}
}
