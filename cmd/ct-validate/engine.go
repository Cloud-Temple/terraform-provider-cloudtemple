package main

import (
	"context"
	"sync"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// EngineConfig parametrizes one engine run.
type EngineConfig struct {
	Runs        int
	Concurrency int
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
func (e *Engine) Run(ctx context.Context, c *client.Client, selected []Cycle, gatedSkips []string) EngineResult {
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
				run := &Run{
					Recorder:  e.rec,
					Breaker:   e.breaker,
					Cleanup:   e.cleanup,
					Iteration: t.iter,
					Worker:    worker,
				}
				// A cycle-level error is intentionally swallowed here: per-op
				// failures are already recorded inside the cycle via run.op,
				// and the breaker decides whether to keep scheduling. The
				// cycle returning an error just means "this iteration aborted
				// early"; the recorded ops carry the detail.
				_ = t.cycle.Run(ctx, c, run)
			}
		}(w)
	}

	wg.Wait()

	// Always tear down what was created, even on a trip or timeout. Give
	// teardown a context that is NOT already cancelled by a breaker trip, but
	// IS still bounded: if the parent ctx is cancelled (global timeout), derive
	// a fresh background-rooted context so best-effort cleanup can still run a
	// bounded number of attempts rather than instantly failing on ctx.Err().
	teardownCtx := ctx
	if ctx.Err() != nil {
		teardownCtx = context.Background()
	}
	teardownFailures := e.cleanup.TeardownAll(teardownCtx)

	names := make([]string, 0, len(selected))
	for _, c := range selected {
		names = append(names, c.Name())
	}

	return EngineResult{
		Stats:           Aggregate(e.rec.Ops()),
		Tripped:         e.breaker.Tripped(),
		TripReason:      e.breaker.Reason(),
		TeardownFailed:  teardownFailures,
		SelectedCycles:  names,
		GatedWriteSkips: gatedSkips,
	}
}
