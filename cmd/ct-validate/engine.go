package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// defaultCleanupTimeout bounds the whole teardown phase when the parent context
// is already cancelled (global -timeout fired). Without a bound, a teardown
// that polls an activity (WaitForCompletion is context-bounded only) could keep
// hitting a distressed API indefinitely on a fresh background context. This is
// the F4 guard: cleanup is best-effort AND bounded.
const defaultCleanupTimeout = 2 * time.Minute

// EngineConfig parametrizes one engine run.
type EngineConfig struct {
	Runs        int
	Concurrency int
	// CleanupTimeout bounds the teardown phase. A non-positive value falls back
	// to defaultCleanupTimeout (applied in NewEngine).
	CleanupTimeout time.Duration
}

// EngineResult summarizes an engine run for the report layer.
type EngineResult struct {
	Stats           []EndpointStats
	Tripped         bool
	TripReason      string
	TeardownFailed  []TeardownFailure
	SelectedCycles  []string
	GatedWriteSkips []string
}

// task is one (cycle, iteration) unit of work for the worker pool.
type task struct {
	cycle Cycle
	iter  int
}

// Engine runs the selected cycles across a bounded worker pool, gated by the
// circuit breaker, and always tears down at the end.
type Engine struct {
	cfg     EngineConfig
	breaker *Breaker
	rec     *Recorder
	cleanup *Cleanup
}

// NewEngine wires an engine to its collaborators.
func NewEngine(cfg EngineConfig, breaker *Breaker, rec *Recorder, cleanup *Cleanup) *Engine {
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.Runs < 1 {
		cfg.Runs = 1
	}
	if cfg.CleanupTimeout <= 0 {
		cfg.CleanupTimeout = defaultCleanupTimeout
	}
	return &Engine{cfg: cfg, breaker: breaker, rec: rec, cleanup: cleanup}
}

// Run executes `Runs` iterations of each selected cycle across `Concurrency`
// workers. Before each task a worker checks the breaker; once it trips, no new
// task is started, the pool drains, and the function returns. Cleanup ALWAYS
// runs (deferred) so nothing created is left orphaned even on a trip or a
// global-timeout cancellation.
//
// The global -timeout is carried by ctx; when it fires, in-flight client calls
// observe context cancellation (CategoryTimeout) and the queue drains.
func (e *Engine) Run(ctx context.Context, c *client.Client, selected []Cycle, gatedSkips []string) (result EngineResult) {
	names := make([]string, 0, len(selected))
	for _, sc := range selected {
		names = append(names, sc.Name())
	}

	// F2/F4: cleanup ALWAYS runs (deferred), even on an early return or a panic
	// that escapes wg.Wait(), and it runs under a FRESH but BOUNDED context so a
	// global-timeout cancellation cannot let teardown poll a distressed API
	// indefinitely. The deferred closure also assembles the result so every exit
	// path reports a well-formed EngineResult.
	defer func() {
		tctx, cancel := e.teardownContext(ctx)
		defer cancel()
		teardownFailures := e.cleanup.TeardownAll(tctx)
		result = EngineResult{
			Stats:           Aggregate(e.rec.Ops()),
			Tripped:         e.breaker.Tripped(),
			TripReason:      e.breaker.Reason(),
			TeardownFailed:  teardownFailures,
			SelectedCycles:  names,
			GatedWriteSkips: gatedSkips,
		}
	}()

	tasks := make(chan task)
	var wg sync.WaitGroup

	// Producer: enqueue (cycle, iter) tasks, stopping as soon as the breaker
	// trips. A read-only default run thus still benefits from the breaker if a
	// listing endpoint starts failing en masse.
	go func() {
		defer close(tasks)
		for iter := 0; iter < e.cfg.Runs; iter++ {
			for _, cyc := range selected {
				if !e.breaker.Allow() {
					return
				}
				select {
				case <-ctx.Done():
					return
				case tasks <- task{cycle: cyc, iter: iter}:
				}
			}
		}
	}()

	// Worker pool.
	for w := 0; w < e.cfg.Concurrency; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for t := range tasks {
				// Re-check at the consumption point: a task already queued
				// before the trip must not run once the breaker is open.
				if !e.breaker.Allow() || ctx.Err() != nil {
					continue
				}
				e.runOne(ctx, c, t, worker)
			}
		}(w)
	}

	wg.Wait()

	// The deferred closure performs teardown and assembles the result.
	return
}

// runOne executes a single (cycle, iteration) task. It is wrapped in a
// per-task recover so a panic in one cycle goroutine neither crashes the whole
// run nor leaves the engine scheduling more work: the panic is recorded as a
// failure op AND trips the breaker (so the producer stops enqueueing and the
// pool drains to the deferred teardown). The created resources of OTHER tasks
// are still torn down by the engine's deferred TeardownAll.
func (e *Engine) runOne(ctx context.Context, c *client.Client, t task, worker int) {
	run := &Run{
		Recorder:  e.rec,
		Breaker:   e.breaker,
		Cleanup:   e.cleanup,
		Iteration: t.iter,
		Worker:    worker,
	}
	defer func() {
		if rec := recover(); rec != nil {
			// Record the panic as a failure outcome so the report shows it, and
			// trip the breaker so no further work is scheduled. CategoryOther is
			// the failure bucket for a non-status error.
			e.rec.Record(Op{
				Cycle:    t.cycle.Name(),
				Endpoint: t.cycle.Name() + ".panic",
				OK:       false,
				Category: CategoryOther,
			})
			e.breaker.Trip(fmt.Sprintf("cycle %q panicked: %v", t.cycle.Name(), rec))
		}
	}()
	// A cycle-level error is intentionally swallowed here: per-op failures are
	// already recorded inside the cycle via run.op, and the breaker decides
	// whether to keep scheduling. The cycle returning an error just means "this
	// iteration aborted early"; the recorded ops carry the detail.
	_ = t.cycle.Run(ctx, c, run)
}

// teardownContext returns the context under which teardown runs, plus a cancel
// the caller MUST defer. Teardown must not be cancelled by a breaker trip (the
// breaker does not cancel ctx), but it MUST stay bounded: if the parent ctx is
// already cancelled (global timeout), derive a FRESH context rooted at
// Background but capped at CleanupTimeout so best-effort cleanup runs a bounded
// number of attempts rather than either failing instantly on ctx.Err() or
// polling a distressed API forever.
//
// When the parent ctx is still live, teardown reuses it (its remaining global
// budget already bounds the work) and the returned cancel is a no-op.
func (e *Engine) teardownContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx.Err() == nil {
		return ctx, func() {}
	}
	return context.WithTimeout(context.Background(), e.cfg.CleanupTimeout)
}
