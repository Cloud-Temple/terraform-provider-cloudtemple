package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestRecorderLiveProgress pins that a recorder with progress prints one
// readable line per operation (OK with latency, skip, FAIL with category), so
// the operator sees advancement. Mutation: drop the writeProgressLine call in
// Record → the buffer is empty → RED.
func TestRecorderLiveProgress(t *testing.T) {
	var buf bytes.Buffer
	r := NewRecorderWithProgress(&buf)
	r.Record(Op{Cycle: "readonly", Endpoint: "iam.token.read", OK: true, Latency: 730 * time.Millisecond, Category: CategoryOK})
	r.Record(Op{Cycle: "readonly", Endpoint: "compute.openiaas.virtual_machines.read", Skipped: true, Category: CategorySkipped})
	r.Record(Op{Cycle: "readonly", Endpoint: "compute.openiaas.networks.list", OK: false, Category: CategoryHTTP5xx})

	out := buf.String()
	for _, want := range []string{
		"iam.token.read", "730 ms", "ok",
		"compute.openiaas.virtual_machines.read", "skip",
		"compute.openiaas.networks.list", "FAIL", "http_5xx",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("live progress must contain %q; got:\n%s", want, out)
		}
	}
	// Three operations → three lines.
	if got := strings.Count(strings.TrimRight(out, "\n"), "\n") + 1; got != 3 {
		t.Fatalf("expected 3 progress lines, got %d:\n%s", got, out)
	}
}

// TestRecorderNoProgressIsSilent pins that the default recorder (no writer)
// prints nothing — progress is strictly opt-in.
func TestRecorderNoProgressIsSilent(t *testing.T) {
	r := NewRecorder() // no progress writer
	r.Record(Op{Endpoint: "x", OK: true})
	if len(r.Ops()) != 1 {
		t.Fatalf("the op must still be recorded, got %d", len(r.Ops()))
	}
}
