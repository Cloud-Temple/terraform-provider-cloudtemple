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

func TestPrintJSONShape(t *testing.T) {
	res := EngineResult{
		SelectedCycles:  []string{"readonly"},
		GatedWriteSkips: []string{"vpc"},
		Stats: []EndpointStats{
			{Cycle: "readonly", Endpoint: "iam.users.list", Total: 2, OK: 1, Reasons: map[Category]int{CategoryHTTP5xx: 1, CategoryOK: 1}},
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
	if len(rep.GatedWrite) != 1 || rep.GatedWrite[0] != "vpc" {
		t.Fatalf("JSON gated_write_cycles wrong: %+v", rep.GatedWrite)
	}
}
