package main

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// Op is a single recorded operation: one call of one endpoint inside one
// cycle iteration. It is the atomic unit the recorder aggregates.
type Op struct {
	Cycle    string
	Endpoint string
	OK       bool
	Skipped  bool
	Latency  time.Duration
	Category Category
}

// Recorder collects Op records concurrently and aggregates them per
// (cycle, endpoint). It is safe for concurrent use by the worker pool.
//
// The aggregation maths (success rate, percentiles, category histogram) is
// PURE: it depends only on the recorded slice, never on wall-clock time or
// network state, so it is fully unit-testable offline.
type Recorder struct {
	mu  sync.Mutex
	ops []Op
	// progress, when set, receives one human-readable line per recorded
	// operation as it happens (live advancement). Writing happens under the
	// lock so concurrent workers never interleave a line.
	progress io.Writer
}

// NewRecorder returns an empty recorder (no live progress).
func NewRecorder() *Recorder {
	return &Recorder{}
}

// NewRecorderWithProgress returns a recorder that also prints one line per
// operation to w as it is recorded, so the operator sees what is running.
func NewRecorderWithProgress(w io.Writer) *Recorder {
	return &Recorder{progress: w}
}

// Record appends a single operation. Safe for concurrent callers.
func (r *Recorder) Record(op Op) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ops = append(r.ops, op)
	if r.progress != nil {
		writeProgressLine(r.progress, len(r.ops), op)
	}
}

// writeProgressLine renders one live advancement line:
//
//	12  ok    iam.token.read                          730 ms
//	18  skip  compute.openiaas.virtual_machines.read
//	21  FAIL  compute.openiaas.networks.list          http_5xx
func writeProgressLine(w io.Writer, n int, op Op) {
	tag, extra := "ok", ""
	switch {
	case op.Skipped:
		tag = "skip"
	case !op.OK:
		tag, extra = "FAIL", string(op.Category)
	case op.Latency > 0:
		extra = fmt.Sprintf("%d ms", op.Latency.Milliseconds())
	}
	fmt.Fprintf(w, "  %3d  %-4s  %-44s  %s\n", n, tag, op.Endpoint, extra)
}

// Ops returns a copy of the recorded operations, in insertion order.
func (r *Recorder) Ops() []Op {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Op, len(r.ops))
	copy(out, r.ops)
	return out
}

// EndpointStats is the aggregate for one (cycle, endpoint) pair.
type EndpointStats struct {
	Cycle    string           `json:"cycle"`
	Endpoint string           `json:"endpoint"`
	Total    int              `json:"total"`
	OK       int              `json:"ok"`
	Skipped  int              `json:"skipped"`
	P50      time.Duration    `json:"p50_ns"`
	P95      time.Duration    `json:"p95_ns"`
	Reasons  map[Category]int `json:"reasons"`
}

// SuccessRate is the fraction of non-skipped attempts that succeeded, in
// [0,1]. Skipped operations are excluded from both numerator and denominator
// so a deliberately skipped sub-step never looks like a failure. When every
// attempt was skipped, the rate is 1 (nothing failed).
func (s EndpointStats) SuccessRate() float64 {
	attempted := s.Total - s.Skipped
	if attempted <= 0 {
		return 1
	}
	return float64(s.OK) / float64(attempted)
}

// percentile returns the p-th percentile (0..100) of durations using the
// nearest-rank method on a sorted copy. It is the single source of truth for
// p50/p95 so the engine and the report can never disagree.
//
// Nearest-rank: rank = ceil(p/100 * n), clamped to [1, n]; the value at that
// 1-based rank is returned. For an empty input it returns 0.
func percentile(durations []time.Duration, p float64) time.Duration {
	n := len(durations)
	if n == 0 {
		return 0
	}
	sorted := make([]time.Duration, n)
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[n-1]
	}
	// Nearest-rank: ceil(p/100 * n) without floating-point rounding surprises.
	rank := int((p*float64(n) + 99.999999) / 100)
	if rank < 1 {
		rank = 1
	}
	if rank > n {
		rank = n
	}
	return sorted[rank-1]
}

// Aggregate folds the recorded operations into per-(cycle,endpoint) stats.
// Latency percentiles are computed only over OK operations (a failed call's
// latency is noise, not a service-level signal). Skipped operations count in
// Total/Skipped but never in OK and never in the latency sample.
//
// The returned slice is deterministically ordered by cycle then endpoint so
// the table and the JSON output are stable across runs.
func Aggregate(ops []Op) []EndpointStats {
	type key struct{ cycle, endpoint string }
	type bucket struct {
		stats     EndpointStats
		latencies []time.Duration
	}
	buckets := map[key]*bucket{}

	for _, op := range ops {
		k := key{op.Cycle, op.Endpoint}
		b := buckets[k]
		if b == nil {
			b = &bucket{stats: EndpointStats{
				Cycle:    op.Cycle,
				Endpoint: op.Endpoint,
				Reasons:  map[Category]int{},
			}}
			buckets[k] = b
		}
		b.stats.Total++
		if op.Skipped {
			b.stats.Skipped++
		}
		if op.OK {
			b.stats.OK++
			b.latencies = append(b.latencies, op.Latency)
		}
		// Record the category for every op (including OK→CategoryOK and
		// Skipped→CategorySkipped) so the histogram is complete.
		b.stats.Reasons[op.Category]++
	}

	out := make([]EndpointStats, 0, len(buckets))
	for _, b := range buckets {
		b.stats.P50 = percentile(b.latencies, 50)
		b.stats.P95 = percentile(b.latencies, 95)
		out = append(out, b.stats)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cycle != out[j].Cycle {
			return out[i].Cycle < out[j].Cycle
		}
		return out[i].Endpoint < out[j].Endpoint
	})
	return out
}
