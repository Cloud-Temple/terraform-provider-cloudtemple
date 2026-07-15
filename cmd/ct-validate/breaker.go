package main

import (
	"fmt"
	"sync"
)

// Breaker is the safety core of the harness. It exists because of a concrete
// incident (2026-06-15): a high-frequency loop against the shared recette API
// AMPLIFIED an outage and produced orphan resources. The breaker stops the
// engine from launching new work the moment the API shows distress.
//
// It trips when EITHER:
//   - consecutive failures reach maxConsecutive, OR
//   - the failure rate over the last `window` recorded outcomes reaches
//     failureRate (evaluated only once the window is full).
//
// Once tripped it stays tripped (latching): Allow() returns false so the
// engine drains and tears down. It is safe for concurrent use.
type Breaker struct {
	mu sync.Mutex

	maxConsecutive int
	failureRate    float64
	window         int

	consecutive int
	recent      []bool // rolling window of "was a failure", newest at the end
	tripped     bool
	reason      string
}

// NewBreaker builds a breaker. A non-positive maxConsecutive disables the
// consecutive-failure rule; a non-positive window OR a failureRate outside
// (0,1] disables the rate rule. With both rules disabled the breaker never
// trips (it is then a no-op guard, still safe to call).
func NewBreaker(maxConsecutive int, failureRate float64, window int) *Breaker {
	return &Breaker{
		maxConsecutive: maxConsecutive,
		failureRate:    failureRate,
		window:         window,
	}
}

// Allow reports whether the engine may launch more work. It returns false once
// the breaker has tripped.
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.tripped
}

// Trip forces the breaker open with an explicit reason, regardless of the
// rolling-window accounting. It is used for out-of-band failures that are not a
// single op outcome — e.g. a cycle goroutine that PANICKED: the panic is
// recovered, recorded as a failure op, and the breaker is tripped so the engine
// stops scheduling new work and drains to teardown. Latching: a no-op once
// already tripped, so the first reason wins.
func (b *Breaker) Trip(reason string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tripped {
		return
	}
	b.tripped = true
	b.reason = reason
}

// Tripped reports whether the breaker has tripped.
func (b *Breaker) Tripped() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tripped
}

// Reason returns the human-readable trip reason, or "" if not tripped.
func (b *Breaker) Reason() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.reason
}

// Record feeds one operation outcome (true = failure) and re-evaluates the
// trip conditions. A skipped op MUST NOT be recorded here: it is neither a
// success nor a failure and would otherwise dilute the rolling window.
//
// Once tripped, further records are ignored (the verdict latches) so a late
// success cannot silently re-arm the engine.
func (b *Breaker) Record(failure bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.tripped {
		return
	}

	if failure {
		b.consecutive++
	} else {
		b.consecutive = 0
	}

	// Maintain the rolling window of the last `window` outcomes.
	if b.window > 0 {
		b.recent = append(b.recent, failure)
		if len(b.recent) > b.window {
			b.recent = b.recent[len(b.recent)-b.window:]
		}
	}

	// Rule (a): consecutive failures.
	if b.maxConsecutive > 0 && b.consecutive >= b.maxConsecutive {
		b.tripped = true
		b.reason = fmt.Sprintf("%d consecutive failures (limit %d)", b.consecutive, b.maxConsecutive)
		return
	}

	// Rule (b): failure rate over a FULL window. Evaluating only on a full
	// window avoids tripping on a single early failure (1/1 = 100%).
	if b.window > 0 && b.failureRate > 0 && len(b.recent) >= b.window {
		failures := 0
		for _, f := range b.recent {
			if f {
				failures++
			}
		}
		rate := float64(failures) / float64(len(b.recent))
		if rate >= b.failureRate {
			b.tripped = true
			b.reason = fmt.Sprintf("failure rate %.0f%% over last %d ops (limit %.0f%%)",
				rate*100, len(b.recent), b.failureRate*100)
		}
	}
}
