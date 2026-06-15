package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// countingCycle records how many times Run was invoked and what each op
// returns. It ignores the client (it makes no network calls), so the engine
// can be unit-tested offline with a nil client.
type countingCycle struct {
	name  string
	kind  Kind
	runs  *int32
	opErr error // returned by the single recorded op per Run
}

func (c countingCycle) Name() string { return c.name }
func (c countingCycle) Kind() Kind   { return c.kind }
func (c countingCycle) Run(ctx context.Context, _ *client.Client, r *Run) error {
	atomic.AddInt32(c.runs, 1)
	return r.op(c, c.name+".op", func() error { return c.opErr })
}

// TestEngineBreakerStopsScheduling proves the breaker stops the engine from
// running every iteration once it trips. With a cycle that always fails and a
// low consecutive limit, far fewer than `runs` iterations execute.
func TestEngineBreakerStopsScheduling(t *testing.T) {
	var runs int32
	cyc := countingCycle{name: "failing", kind: KindRead, runs: &runs, opErr: client.StatusError{Code: 500}}

	// Trip after 3 consecutive failures; single worker for determinism.
	breaker := NewBreaker(3, 1.0, 1000)
	rec := NewRecorder()
	cleanup := NewCleanup()
	eng := NewEngine(EngineConfig{Runs: 100, Concurrency: 1}, breaker, rec, cleanup)

	res := eng.Run(context.Background(), nil, []Cycle{cyc}, nil)

	if !res.Tripped {
		t.Fatal("breaker should have tripped on a constantly-failing cycle")
	}
	// The breaker trips at 3 consecutive failures. The engine gates work in TWO
	// places (defense in depth): the producer stops ENQUEUEING once Allow() is
	// false, and each worker RE-CHECKS before running a task that was already
	// queued when the trip happened. Together they guarantee work stops almost
	// immediately after the trip — FAR below the 100 requested iterations.
	// Removing BOTH guards lets all 100 run (the mutation-proof for this test).
	// With concurrency=1 and a trip at 3, only a handful of cycles run.
	if got := atomic.LoadInt32(&runs); got >= 10 {
		t.Fatalf("breaker did not stop scheduling promptly: ran %d/100 iterations", got)
	}
	if got := atomic.LoadInt32(&runs); got < 3 {
		t.Fatalf("expected at least 3 runs before trip, ran %d", got)
	}
}

// TestEngineAlwaysTearsDown proves cleanup ALWAYS runs at the end, even when
// the breaker trips mid-run: a resource registered by an early iteration must
// be torn down.
func TestEngineAlwaysTearsDown(t *testing.T) {
	var torn int32
	registering := registerThenFailCycle{torn: &torn}

	breaker := NewBreaker(2, 1.0, 1000)
	eng := NewEngine(EngineConfig{Runs: 50, Concurrency: 1}, breaker, NewRecorder(), NewCleanup())
	res := eng.Run(context.Background(), nil, []Cycle{registering}, nil)

	if !res.Tripped {
		t.Fatal("setup: expected the breaker to trip")
	}
	if atomic.LoadInt32(&torn) == 0 {
		t.Fatal("cleanup must run even when the breaker trips mid-run")
	}
}

// registerThenFailCycle registers a teardown, then fails its op, so the breaker
// trips while there is something to clean up.
type registerThenFailCycle struct {
	torn *int32
}

func (registerThenFailCycle) Name() string { return "reg-fail" }
func (registerThenFailCycle) Kind() Kind   { return KindWrite }
func (c registerThenFailCycle) Run(ctx context.Context, _ *client.Client, r *Run) error {
	r.Cleanup.Register("res", func(context.Context) error {
		atomic.AddInt32(c.torn, 1)
		return nil
	})
	return r.op(c, "reg-fail.op", func() error { return client.StatusError{Code: 500} })
}

// TestEngineRespectsGlobalTimeout proves a cancelled context drains the engine
// promptly (the producer/worker observe ctx.Done()) rather than running all
// iterations.
func TestEngineRespectsGlobalTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	var runs int32
	cyc := countingCycle{name: "x", kind: KindRead, runs: &runs, opErr: nil}
	eng := NewEngine(EngineConfig{Runs: 100, Concurrency: 2}, NewBreaker(1000, 0.99, 1000), NewRecorder(), NewCleanup())
	res := eng.Run(ctx, nil, []Cycle{cyc}, nil)

	if got := atomic.LoadInt32(&runs); got >= 100 {
		t.Fatalf("cancelled context should curtail scheduling, ran %d/100", got)
	}
	// Even on cancellation the result is well-formed (not tripped, no panic).
	if res.Tripped {
		t.Fatal("a cancelled context must not be reported as a breaker trip")
	}
}

// TestEngineTeardownReportedWhenCleanupFails proves teardown failures surface
// in the result (so main can exit non-zero on a possible orphan).
func TestEngineTeardownReportedWhenCleanupFails(t *testing.T) {
	failing := failingTeardownCycle{}
	eng := NewEngine(EngineConfig{Runs: 1, Concurrency: 1}, NewBreaker(1000, 0.99, 1000), NewRecorder(), NewCleanup())
	res := eng.Run(context.Background(), nil, []Cycle{failing}, nil)
	if len(res.TeardownFailed) == 0 {
		t.Fatal("a failing teardown must be reported in the engine result")
	}
}

type failingTeardownCycle struct{}

func (failingTeardownCycle) Name() string { return "ft" }
func (failingTeardownCycle) Kind() Kind   { return KindWrite }
func (c failingTeardownCycle) Run(ctx context.Context, _ *client.Client, r *Run) error {
	r.Cleanup.Register("orphan", func(context.Context) error {
		return errors.New("cannot delete")
	})
	return r.op(c, "ft.op", func() error { return nil })
}
