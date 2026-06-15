package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

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

// panickingCycle registers a teardown, then PANICS in Run. It is the F2 fixture:
// the engine must recover, run the teardown, trip the breaker, and report the
// panic as a failure — never crash the process.
type panickingCycle struct {
	torn *int32
}

func (panickingCycle) Name() string { return "boom" }
func (panickingCycle) Kind() Kind   { return KindWrite }
func (c panickingCycle) Run(ctx context.Context, _ *client.Client, r *Run) error {
	r.Cleanup.Register("res", func(context.Context) error {
		atomic.AddInt32(c.torn, 1)
		return nil
	})
	panic("synthetic cycle panic")
}

// TestEnginePanicRecoveredTornDownAndTripped proves F2: a cycle whose Run panics
// does NOT crash the engine. Teardown still runs (deferred), the breaker trips,
// and the panic is recorded as a failure outcome.
//
// Mutation proofs:
//   - remove the per-task recover() in runOne → the panic escapes the worker
//     goroutine → the test process crashes (RED).
//   - remove the breaker.Trip(...) in the recover → res.Tripped is false (RED).
//   - move TeardownAll out of the deferred closure (back after wg.Wait only) →
//     the panic skips it → torn stays 0 (RED).
func TestEnginePanicRecoveredTornDownAndTripped(t *testing.T) {
	var torn int32
	cyc := panickingCycle{torn: &torn}

	breaker := NewBreaker(1000, 0.99, 1000) // would NOT trip on accounting alone
	rec := NewRecorder()
	eng := NewEngine(EngineConfig{Runs: 3, Concurrency: 1}, breaker, rec, NewCleanup())

	res := eng.Run(context.Background(), nil, []Cycle{cyc}, nil) // must not panic out

	if atomic.LoadInt32(&torn) == 0 {
		t.Fatal("teardown must run even when a cycle panics (deferred TeardownAll)")
	}
	if !res.Tripped {
		t.Fatal("a panicking cycle must trip the breaker so scheduling stops")
	}
	// The panic must surface as a recorded failure op (endpoint "<cycle>.panic").
	var sawPanicOp bool
	for _, o := range rec.Ops() {
		if o.Endpoint == "boom.panic" {
			sawPanicOp = true
			if o.OK || o.Category == CategoryOK {
				t.Fatalf("panic op must be a failure, got %+v", o)
			}
		}
	}
	if !sawPanicOp {
		t.Fatal("the panic must be recorded as a failure op for the report")
	}
}

// TestEngineTeardownBoundedOnCancelledContext proves F4: when the parent ctx is
// already cancelled (global -timeout), teardown runs under a FRESH but BOUNDED
// context, NOT context.Background() with no deadline. A teardown that honours
// its context observes a deadline and is cut off rather than polling forever.
//
// Mutation proof: change teardownContext to return context.Background() (no
// timeout) when ctx is cancelled → the teardown sees ctx.Err()==nil and no
// Deadline → this test goes RED.
func TestEngineTeardownBoundedOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // parent already cancelled (simulates global timeout fired)

	cleanup := NewCleanup()
	var sawDeadline int32
	cleanup.Register("bounded", func(tctx context.Context) error {
		// The teardown context must NOT be the cancelled parent (that would fail
		// instantly) and MUST carry a deadline (bounded), not a naked Background.
		if tctx.Err() != nil {
			t.Errorf("teardown context must not be pre-cancelled")
		}
		if _, ok := tctx.Deadline(); ok {
			atomic.StoreInt32(&sawDeadline, 1)
		}
		return nil
	})

	// A tiny cleanup timeout proves the bound is honoured and configurable.
	eng := NewEngine(
		EngineConfig{Runs: 1, Concurrency: 1, CleanupTimeout: 50 * time.Millisecond},
		NewBreaker(1000, 0.99, 1000), NewRecorder(), cleanup,
	)
	_ = eng.Run(ctx, nil, []Cycle{noopCycle{}}, nil)

	if atomic.LoadInt32(&sawDeadline) == 0 {
		t.Fatal("teardown after a cancelled parent must run under a BOUNDED (deadlined) context")
	}
}

// noopCycle does nothing; it lets the engine reach its deferred teardown with a
// pre-registered cleanup entry under test.
type noopCycle struct{}

func (noopCycle) Name() string                                    { return "noop" }
func (noopCycle) Kind() Kind                                      { return KindRead }
func (noopCycle) Run(context.Context, *client.Client, *Run) error { return nil }

// TestEngineUsesParentContextWhenLive proves the symmetric case: when the parent
// ctx is still live, teardown reuses it (no fresh background context), so the
// remaining global budget bounds cleanup.
func TestEngineUsesParentContextWhenLive(t *testing.T) {
	ctx := context.Background() // live, no deadline
	cleanup := NewCleanup()
	var reusedParent int32
	cleanup.Register("x", func(tctx context.Context) error {
		// Background has no deadline; if teardown wrongly wrapped a timeout here,
		// a deadline would appear.
		if _, ok := tctx.Deadline(); !ok {
			atomic.StoreInt32(&reusedParent, 1)
		}
		return nil
	})
	eng := NewEngine(EngineConfig{Runs: 1, Concurrency: 1}, NewBreaker(1000, 0.99, 1000), NewRecorder(), cleanup)
	_ = eng.Run(ctx, nil, []Cycle{noopCycle{}}, nil)
	if atomic.LoadInt32(&reusedParent) == 0 {
		t.Fatal("a live parent context must be reused for teardown, not replaced by a bounded one")
	}
}
