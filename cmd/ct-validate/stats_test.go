package main

import (
	"testing"
	"time"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func TestPercentileNearestRank(t *testing.T) {
	// 1..10 ms. Nearest-rank p50 of 10 = ceil(0.5*10)=5th value = 5ms.
	// p95 = ceil(0.95*10)=10th value = 10ms.
	d := []time.Duration{ms(1), ms(2), ms(3), ms(4), ms(5), ms(6), ms(7), ms(8), ms(9), ms(10)}
	if got := percentile(d, 50); got != ms(5) {
		t.Errorf("p50 = %v, want 5ms", got)
	}
	if got := percentile(d, 95); got != ms(10) {
		t.Errorf("p95 = %v, want 10ms", got)
	}
}

// TestPercentileUnsortedInput proves percentile sorts a copy: an out-of-order
// input must give the same answer as the sorted one, and must NOT mutate the
// caller's slice. The input is chosen so the element at the p50 nearest-rank
// position (index 2) DIFFERS between the unsorted and sorted orders: unsorted
// [10,50,40,20,30][2] = 40ms, but the true p50 is 30ms. A mutation that drops
// the internal sort returns 40ms and turns this RED.
func TestPercentileUnsortedInput(t *testing.T) {
	in := []time.Duration{ms(10), ms(50), ms(40), ms(20), ms(30)}
	orig := append([]time.Duration(nil), in...)
	if got := percentile(in, 50); got != ms(30) {
		t.Errorf("p50 of shuffled = %v, want 30ms (sorted median)", got)
	}
	for i := range in {
		if in[i] != orig[i] {
			t.Fatalf("percentile mutated caller's slice at %d: %v != %v", i, in[i], orig[i])
		}
	}
}

func TestPercentileEdges(t *testing.T) {
	if got := percentile(nil, 50); got != 0 {
		t.Errorf("empty p50 = %v, want 0", got)
	}
	one := []time.Duration{ms(7)}
	if got := percentile(one, 95); got != ms(7) {
		t.Errorf("single-element p95 = %v, want 7ms", got)
	}
	if got := percentile(one, 0); got != ms(7) {
		t.Errorf("p0 = %v, want min 7ms", got)
	}
}

func TestSuccessRate(t *testing.T) {
	cases := []struct {
		name  string
		stats EndpointStats
		want  float64
	}{
		{"all ok", EndpointStats{Total: 4, OK: 4}, 1.0},
		{"half", EndpointStats{Total: 4, OK: 2}, 0.5},
		{"skips excluded", EndpointStats{Total: 4, OK: 2, Skipped: 2}, 1.0}, // 2 ok / 2 attempted
		{"all skipped is 1", EndpointStats{Total: 3, OK: 0, Skipped: 3}, 1.0},
		{"none at all is 1", EndpointStats{}, 1.0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.stats.SuccessRate(); got != c.want {
				t.Fatalf("SuccessRate = %g, want %g", got, c.want)
			}
		})
	}
}

func TestAggregate(t *testing.T) {
	ops := []Op{
		{Cycle: "readonly", Endpoint: "iam.users.list", OK: true, Latency: ms(10), Category: CategoryOK},
		{Cycle: "readonly", Endpoint: "iam.users.list", OK: true, Latency: ms(30), Category: CategoryOK},
		{Cycle: "readonly", Endpoint: "iam.users.list", OK: false, Latency: ms(999), Category: CategoryHTTP5xx},
		{Cycle: "readonly", Endpoint: "vpc.vpc.list", OK: false, Latency: ms(2), Category: CategoryBadGateway502},
		{Cycle: "readonly", Endpoint: "iam.users.read", Skipped: true, Category: CategorySkipped},
	}
	stats := Aggregate(ops)

	// Deterministic ordering: iam.users.list, iam.users.read, vpc.vpc.list.
	if len(stats) != 3 {
		t.Fatalf("want 3 endpoint groups, got %d", len(stats))
	}
	if stats[0].Endpoint != "iam.users.list" || stats[1].Endpoint != "iam.users.read" || stats[2].Endpoint != "vpc.vpc.list" {
		t.Fatalf("aggregate not sorted by cycle/endpoint: %+v", stats)
	}

	list := stats[0]
	if list.Total != 3 || list.OK != 2 {
		t.Fatalf("iam.users.list totals: total=%d ok=%d", list.Total, list.OK)
	}
	// Percentiles computed ONLY over OK latencies (10ms, 30ms), so the failed
	// 999ms must NOT pollute them. A mutation that includes failed latencies
	// would push p95 to 999ms.
	if list.P95 == ms(999) {
		t.Fatal("failed-op latency (999ms) leaked into the OK percentile sample")
	}
	if list.P50 != ms(10) || list.P95 != ms(30) {
		t.Fatalf("OK-only percentiles: p50=%v p95=%v (want 10ms/30ms)", list.P50, list.P95)
	}
	if list.Reasons[CategoryHTTP5xx] != 1 || list.Reasons[CategoryOK] != 2 {
		t.Fatalf("reason histogram wrong: %+v", list.Reasons)
	}

	// Skipped endpoint: Total counts it, Skipped counts it, OK does not.
	read := stats[1]
	if read.Total != 1 || read.Skipped != 1 || read.OK != 0 {
		t.Fatalf("skipped endpoint accounting wrong: %+v", read)
	}
}

func TestHasFailure(t *testing.T) {
	clean := []EndpointStats{{Total: 3, OK: 3}, {Total: 2, OK: 0, Skipped: 2}}
	if HasFailure(clean) {
		t.Fatal("clean stats (all ok or all skipped) must not report failure")
	}
	dirty := []EndpointStats{{Total: 3, OK: 3}, {Total: 2, OK: 1}}
	if !HasFailure(dirty) {
		t.Fatal("a partially-failing endpoint must report failure")
	}
}
