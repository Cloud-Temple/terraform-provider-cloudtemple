package main

import (
	"context"
	"sync"
)

// teardownMaxAttempts bounds the retries of a single teardown closure. The
// first attempt plus (teardownMaxAttempts-1) bounded retries on a TRANSIENT
// failure. This is the never-orphan guarantee without the never-stop hazard
// that caused the 2026-06-15 incident: no infinite retry, ever.
const teardownMaxAttempts = 3

// CleanupFunc tears down one created resource. It receives a context so a
// global timeout still bounds teardown.
type CleanupFunc func(ctx context.Context) error

// cleanupEntry pairs a teardown closure with a label for the failure report.
type cleanupEntry struct {
	label string
	fn    CleanupFunc
}

// Cleanup tracks created resources and tears them down LIFO. It is the
// never-orphan backstop: every write step registers its teardown BEFORE the
// created resource can be lost to the state, so even if a later step or the
// breaker aborts the cycle, TeardownAll still removes what was created.
//
// Safe for concurrent registration by the worker pool.
type Cleanup struct {
	mu      sync.Mutex
	entries []cleanupEntry
}

// NewCleanup returns an empty tracker.
func NewCleanup() *Cleanup {
	return &Cleanup{}
}

// Register adds a teardown closure. Closures run in reverse registration order
// (LIFO) so dependents are removed before their dependencies (e.g. unbind a
// floating IP before deleting the static IP it pointed at).
func (c *Cleanup) Register(label string, fn CleanupFunc) {
	c.mu.Lock()
	c.entries = append(c.entries, cleanupEntry{label: label, fn: fn})
	c.mu.Unlock()
}

// TeardownFailure records a teardown that could not complete.
type TeardownFailure struct {
	Label    string
	Err      error
	Attempts int
}

// TeardownAll runs every registered teardown LIFO, one logical teardown each
// with a bounded transient-retry. A failing teardown NEVER stops the others:
// the goal is best-effort removal of everything that was created. It returns
// the list of teardowns that still failed after their bounded retries.
//
// The tracker is drained as it goes, so calling TeardownAll twice is safe and
// the second call is a no-op (idempotent under the engine's defer + explicit
// call paths).
func (c *Cleanup) TeardownAll(ctx context.Context) []TeardownFailure {
	c.mu.Lock()
	entries := c.entries
	c.entries = nil
	c.mu.Unlock()

	var failures []TeardownFailure
	// LIFO: iterate from the most recently registered to the oldest.
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		attempts, err := runTeardown(ctx, e.fn)
		if err != nil {
			failures = append(failures, TeardownFailure{Label: e.label, Err: err, Attempts: attempts})
		}
	}
	return failures
}

// Pending reports how many teardowns are still registered.
func (c *Cleanup) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// runTeardown executes one teardown with a bounded transient-retry. It returns
// the number of attempts made and the final error (nil on success).
//
// Retry policy: only TRANSIENT categories (transient workers, 502, 5xx) are
// retried; a permanent failure (4xx, decode, etc.) is returned on the first
// attempt. The context is checked between attempts so a global timeout cuts it
// short. There is no sleep: a bounded immediate retry is enough to ride over a
// single transient blip without becoming a load loop.
func runTeardown(ctx context.Context, fn CleanupFunc) (int, error) {
	var lastErr error
	for attempt := 1; attempt <= teardownMaxAttempts; attempt++ {
		if ctx.Err() != nil {
			if lastErr == nil {
				lastErr = ctx.Err()
			}
			return attempt - 1, lastErr
		}
		lastErr = fn(ctx)
		if lastErr == nil {
			return attempt, nil
		}
		if !isTransientCategory(categorize(lastErr)) {
			return attempt, lastErr
		}
	}
	return teardownMaxAttempts, lastErr
}

// isTransientCategory reports whether a category is worth a bounded retry
// during teardown.
func isTransientCategory(cat Category) bool {
	switch cat {
	case CategoryTransientWorkers, CategoryBadGateway502, CategoryHTTP5xx:
		return true
	default:
		return false
	}
}
