package main

import (
	"sync"
	"testing"
)

// feed records a sequence of failure flags into a breaker.
func feed(b *Breaker, failures ...bool) {
	for _, f := range failures {
		b.Record(f)
	}
}

func TestBreakerConsecutiveTrips(t *testing.T) {
	// maxConsecutive=3, rate rule effectively off (rate=1, window large).
	b := NewBreaker(3, 1.0, 100)
	feed(b, true, true)
	if b.Tripped() {
		t.Fatal("must not trip at 2 consecutive (limit 3)")
	}
	if !b.Allow() {
		t.Fatal("Allow must be true before tripping")
	}
	b.Record(true)
	if !b.Tripped() {
		t.Fatal("must trip at 3 consecutive")
	}
	if b.Allow() {
		t.Fatal("Allow must be false after tripping")
	}
}

// TestBreakerConsecutiveResetBySuccess proves the consecutive counter resets on
// a success: 2 fails, a success, 2 fails must NOT trip a limit-3 breaker. A
// mutation that fails to reset b.consecutive on success would make this RED.
func TestBreakerConsecutiveResetBySuccess(t *testing.T) {
	b := NewBreaker(3, 1.0, 100)
	feed(b, true, true, false, true, true)
	if b.Tripped() {
		t.Fatal("success between failures must reset the consecutive counter")
	}
}

func TestBreakerRateTripsOnFullWindow(t *testing.T) {
	// window=10, rate=0.30 → trips when >=3 of the last 10 ops are failures.
	// maxConsecutive is large so ONLY the rate rule is under test here.
	b := NewBreaker(1000, 0.30, 10)
	// Fill the window with successes (0% over a full window).
	for i := 0; i < 10; i++ {
		b.Record(false)
	}
	// Two trailing failures → window = [S×8, F, F] = 2/10 = 20%: must NOT trip.
	feed(b, true, true)
	if b.Tripped() {
		t.Fatalf("20%% failure must not trip a 30%% breaker; reason=%q", b.Reason())
	}
	// A third trailing failure → window = [S×7, F, F, F] = 3/10 = 30% ≥ limit.
	b.Record(true)
	if !b.Tripped() {
		t.Fatal("30%% failure over a full window must trip a 30%% breaker")
	}
}

// TestBreakerRateNeedsFullWindow proves the rate rule does NOT evaluate before
// the window is full: a single first-op failure is 1/1 = 100% but must not
// trip. A mutation that drops the `len(recent) >= window` guard turns this RED.
func TestBreakerRateNeedsFullWindow(t *testing.T) {
	b := NewBreaker(1000, 0.30, 10)
	b.Record(true)
	if b.Tripped() {
		t.Fatal("a single early failure must not trip the rate rule before the window is full")
	}
}

// TestBreakerRollingWindowForgetsOldFailures proves the window truly ROLLS:
// old failures must age out so the rate is computed over the LAST `window`
// ops, not over all history. The threshold (0.75) is chosen so the verdict
// differs between a correctly-trimmed window and an untrimmed one:
//
//   - trimmed (correct): during phase 1 no 4-op window holds 3 failures, so the
//     rate never reaches 75% and the breaker does not trip; then 3 trailing
//     failures make the last-4 window [S,F,F,F] = 75% ≥ limit → trips.
//   - untrimmed (mutated, no `recent = recent[len(recent)-window:]`): the rate
//     would be computed over all 11 ops = 5/11 ≈ 45% < 75% → would NOT trip,
//     turning the second assertion RED.
//
// maxConsecutive is large so only the rate rule is exercised.
func TestBreakerRollingWindowForgetsOldFailures(t *testing.T) {
	b := NewBreaker(1000, 0.75, 4)
	// Phase 1: 2 sparse failures then successes age them out. No 4-op window
	// ever holds 3 failures, so the 75% rate is never reached here.
	feed(b, true, false, false, true, false, false, false, false)
	if b.Tripped() {
		t.Fatalf("old failures must roll out of the window before tripping; reason=%q", b.Reason())
	}
	// Phase 2: 3 trailing failures → trimmed last-4 window = [S,F,F,F] = 75%.
	feed(b, true, true, true)
	if !b.Tripped() {
		t.Fatal("75%% over the last 4 (trimmed) window must trip a 75%% breaker")
	}
}

// TestBreakerLatches proves the verdict latches: once tripped, the breaker
// stops re-evaluating. Two observable consequences are asserted:
//   - a flood of successes must NOT re-arm the engine; and
//   - the trip REASON is FROZEN at the moment of tripping (the operator must
//     see WHY it first tripped, not a later, larger count).
//
// The frozen-reason assertion is the mutation-sensitive one: removing the early
// `if b.tripped { return }` in Record lets later failures keep recomputing and
// rewrite the reason (e.g. "2 consecutive" → "4 consecutive"), turning this RED.
func TestBreakerLatches(t *testing.T) {
	b := NewBreaker(2, 1.0, 100)
	feed(b, true, true) // trips on 2 consecutive
	if !b.Tripped() {
		t.Fatal("setup: expected trip")
	}
	reasonAtTrip := b.Reason()

	// More failures after the trip must not change the verdict OR the reason.
	feed(b, true, true)
	if b.Reason() != reasonAtTrip {
		t.Fatalf("trip reason mutated after latching: %q -> %q", reasonAtTrip, b.Reason())
	}

	// A success flood must not re-arm the breaker.
	feed(b, false, false, false, false)
	if !b.Tripped() || b.Allow() {
		t.Fatal("breaker must stay tripped after a success flood")
	}
}

// TestBreakerDisabledRules proves a fully-disabled breaker never trips and is a
// safe no-op guard.
func TestBreakerDisabledRules(t *testing.T) {
	b := NewBreaker(0, 0, 0)
	for i := 0; i < 100; i++ {
		b.Record(true)
	}
	if b.Tripped() || !b.Allow() {
		t.Fatal("a disabled breaker must never trip")
	}
}

// TestBreakerConcurrentRecord is a race-detector smoke test: concurrent
// Record/Allow must not data-race.
func TestBreakerConcurrentRecord(t *testing.T) {
	b := NewBreaker(1000, 0.99, 1000)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				b.Record(i%2 == 0)
				_ = b.Allow()
			}
		}()
	}
	wg.Wait()
}
