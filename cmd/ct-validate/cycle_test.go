package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// fakeCycle is a no-op cycle for registry/selection tests.
type fakeCycle struct {
	name string
	kind Kind
}

func (f fakeCycle) Name() string                                    { return f.name }
func (f fakeCycle) Kind() Kind                                      { return f.kind }
func (f fakeCycle) Run(context.Context, *client.Client, *Run) error { return nil }

// quarantinedFake is a write cycle that opts out of the "all" selector, the way
// the real vpcCycle does. It embeds fakeCycle (inheriting Name/Kind/Run) and
// only adds the Quarantined capability.
type quarantinedFake struct{ fakeCycle }

func (quarantinedFake) Quarantined() bool { return true }

func names(cs []Cycle) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Name()
	}
	return out
}

func testRegistry() *Registry {
	return NewRegistry(
		fakeCycle{"readonly", KindRead},
		fakeCycle{"backup", KindRead},
		// "vpc" mirrors the real vpcCycle: a quarantined write cycle, excluded
		// from "all" and reachable only by naming it explicitly.
		quarantinedFake{fakeCycle{"vpc", KindWrite}},
		fakeCycle{"object_storage", KindWrite},
	)
}

func TestSelectCSV(t *testing.T) {
	reg := testRegistry()
	sel, gated, err := reg.Select("readonly, backup", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"readonly", "backup"}) {
		t.Fatalf("selection = %v, want [readonly backup]", got)
	}
	if len(gated) != 0 {
		t.Fatalf("no write cycles selected, gated should be empty: %v", gated)
	}
}

// TestSelectTrimsAndDropsEmptyTokens proves whitespace is trimmed and empty
// tokens are ignored (a trailing comma must not become an unknown "" cycle).
func TestSelectTrimsAndDropsEmptyTokens(t *testing.T) {
	reg := testRegistry()
	sel, _, err := reg.Select("  readonly ,, backup ,", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"readonly", "backup"}) {
		t.Fatalf("selection = %v, want [readonly backup]", got)
	}
}

// TestSelectDeduplicates proves a repeated name collapses to one entry,
// first-seen order preserved.
func TestSelectDeduplicates(t *testing.T) {
	reg := testRegistry()
	sel, _, err := reg.Select("backup,readonly,backup", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"backup", "readonly"}) {
		t.Fatalf("selection = %v, want [backup readonly] (deduped, first-seen order)", got)
	}
}

func TestSelectAll(t *testing.T) {
	reg := testRegistry()
	// With -write, "all" includes the write cycles, ordered by name — EXCEPT a
	// quarantined cycle ("vpc"), which "all" must never expand to.
	sel, gated, err := reg.Select("all", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"backup", "object_storage", "readonly"}) {
		t.Fatalf("all (write) = %v, want [backup object_storage readonly] (vpc is quarantined)", got)
	}
	if len(gated) != 0 {
		t.Fatalf("write enabled, nothing should be gated: %v", gated)
	}
}

// TestSelectAllSupersedes proves "all" combined with explicit names is still
// just "all": the full UNQUARANTINED set, deduplicated. A quarantined cycle
// ("vpc") named alongside "all" is still excluded — fail closed, so a blanket
// sweep cannot be tricked into firing it by also naming it.
func TestSelectAllSupersedes(t *testing.T) {
	reg := testRegistry()
	sel, _, err := reg.Select("vpc,all,readonly", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"backup", "object_storage", "readonly"}) {
		t.Fatalf("all-supersedes = %v, want [backup object_storage readonly] (vpc quarantined out of all)", got)
	}
}

// TestSelectWriteGating proves write cycles are dropped (and reported as gated)
// when -write is false, while read cycles still run. A mutation that ignores
// the write flag would let a write cycle through here.
func TestSelectWriteGating(t *testing.T) {
	reg := testRegistry()
	sel, gated, err := reg.Select("readonly,vpc,object_storage", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"readonly"}) {
		t.Fatalf("write-gated selection = %v, want [readonly]", got)
	}
	if !reflect.DeepEqual(gated, []string{"vpc", "object_storage"}) {
		t.Fatalf("gated = %v, want [vpc object_storage]", gated)
	}
}

// TestSelectAllWriteGating proves "all" without -write yields only the read
// cycles, with the (unquarantined) write cycles reported as gated. The
// quarantined "vpc" cycle is excluded from "all" entirely, so it is neither
// selected nor reported as gated.
func TestSelectAllWriteGating(t *testing.T) {
	reg := testRegistry()
	sel, gated, err := reg.Select("all", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"backup", "readonly"}) {
		t.Fatalf("all (read-only) = %v, want [backup readonly]", got)
	}
	if !reflect.DeepEqual(gated, []string{"object_storage"}) {
		t.Fatalf("gated = %v, want [object_storage] (vpc is quarantined out of all, not gated)", gated)
	}
}

// TestSelectAllExcludesQuarantinedButExplicitRuns locks the quarantine
// invariant at the selector level: a blanket "all -write" must NOT expand to a
// quarantined cycle, but naming it explicitly still runs it. Quarantine removes
// a cycle from the sweep; it does not disable the cycle.
func TestSelectAllExcludesQuarantinedButExplicitRuns(t *testing.T) {
	reg := NewRegistry(
		fakeCycle{"readonly", KindRead},
		quarantinedFake{fakeCycle{"vpc", KindWrite}},
	)

	// "all -write" must exclude the quarantined cycle.
	sel, _, err := reg.Select("all", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"readonly"}) {
		t.Fatalf("all (write) = %v, want [readonly] (vpc quarantined out of all)", got)
	}

	// Naming it explicitly (with -write) still runs it.
	sel, gated, err := reg.Select("vpc", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"vpc"}) {
		t.Fatalf("explicit vpc = %v, want [vpc]", got)
	}
	if len(gated) != 0 {
		t.Fatalf("write enabled, nothing should be gated: %v", gated)
	}
}

// TestBuildRegistryQuarantinesVPC proves the REAL registry wiring: the shipped
// vpcCycle declares itself quarantined, so the production `-cycles all -write`
// path can never fire the deprecated /vpc/v1 write cycle, while `-cycles vpc
// -write` still does. Without vpcCycle.Quarantined()==true this fails.
func TestBuildRegistryQuarantinesVPC(t *testing.T) {
	reg := buildRegistry()

	sel, _, err := reg.Select("all", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, n := range names(sel) {
		if n == "vpc" {
			t.Fatalf(`"all" -write expanded to include "vpc"; the quarantined /vpc/v1 cycle must be excluded from "all" (got %v)`, names(sel))
		}
	}

	sel, _, err = reg.Select("vpc", true)
	if err != nil {
		t.Fatalf("unexpected error selecting explicit vpc: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"vpc"}) {
		t.Fatalf("explicit vpc = %v, want [vpc] (quarantine must not disable explicit selection)", got)
	}
}

func TestSelectUnknownIsError(t *testing.T) {
	reg := testRegistry()
	if _, _, err := reg.Select("readonly,does_not_exist", true); err == nil {
		t.Fatal("an unknown cycle name must be a hard error, not a silent skip")
	}
}

func TestSelectEmptyIsError(t *testing.T) {
	reg := testRegistry()
	if _, _, err := reg.Select("   ,, ", true); err == nil {
		t.Fatal("an all-empty spec must be an error")
	}
}

// TestRegistryDuplicatePanics proves the registry rejects duplicate names
// (programmer error caught early).
func TestRegistryDuplicatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("registering two cycles with the same name must panic")
		}
	}()
	NewRegistry(fakeCycle{"dup", KindRead}, fakeCycle{"dup", KindWrite})
}

// TestRunOpRecordsAndFeedsBreaker proves the op() choke point records an Op AND
// feeds the breaker: a failure must increment failure accounting; an OK must
// reset consecutive; a skip must touch neither the breaker nor count as OK.
func TestRunOpRecordsAndFeedsBreaker(t *testing.T) {
	rec := NewRecorder()
	b := NewBreaker(2, 1.0, 100)
	r := &Run{Recorder: rec, Breaker: b, Cleanup: NewCleanup()}
	c := fakeCycle{"readonly", KindRead}

	_ = r.op(c, "x.list", func() error { return nil })                           // ok
	_ = r.op(c, "x.list", func() error { return client.StatusError{Code: 500} }) // fail 1
	r.skip(c, "x.read")                                                          // skip: not a breaker failure
	if b.Tripped() {
		t.Fatal("one failure with a skip in between must not trip a limit-2 breaker")
	}
	_ = r.op(c, "x.list", func() error { return client.StatusError{Code: 500} }) // fail 2
	if !b.Tripped() {
		t.Fatal("two failures must trip the limit-2 breaker (skip must not have reset it)")
	}

	ops := rec.Ops()
	if len(ops) != 4 {
		t.Fatalf("recorded %d ops, want 4", len(ops))
	}
	// The skip must be marked skipped and not OK.
	var skip Op
	for _, o := range ops {
		if o.Endpoint == "x.read" {
			skip = o
		}
	}
	if !skip.Skipped || skip.OK || skip.Category != CategorySkipped {
		t.Fatalf("skip op recorded wrong: %+v", skip)
	}
}

// TestRunOpGatesOnTrippedBreaker proves F1: once the breaker is tripped, op()
// does NOT call fn — it records a skip and returns nil. This is what bounds the
// hammering MID-CYCLE (a long cycle must not keep calling the API after an early
// trip).
//
// Mutation proof: remove the `if r.Breaker != nil && !r.Breaker.Allow()` gate at
// the top of Run.op → fn IS called → called becomes true → this test goes RED.
func TestRunOpGatesOnTrippedBreaker(t *testing.T) {
	rec := NewRecorder()
	b := NewBreaker(1, 1.0, 100)
	b.Trip("pre-tripped for test") // force the breaker open before any op
	r := &Run{Recorder: rec, Breaker: b, Cleanup: NewCleanup()}
	c := fakeCycle{"readonly", KindRead}

	called := false
	got := r.op(c, "x.list", func() error { called = true; return nil })

	if called {
		t.Fatal("op must NOT call fn once the breaker has tripped (mid-cycle gating)")
	}
	if got != nil {
		t.Fatalf("a gated op must return nil, got %v", got)
	}
	ops := rec.Ops()
	if len(ops) != 1 {
		t.Fatalf("recorded %d ops, want exactly 1 (the skip)", len(ops))
	}
	o := ops[0]
	if !o.Skipped || o.OK || o.Category != CategorySkipped || o.Endpoint != "x.list" {
		t.Fatalf("gated op must be recorded as a skip, got %+v", o)
	}
}

// TestRunOpSkipDoesNotFeedBreaker proves a gated skip does not feed the breaker
// window: a tripped breaker stays tripped on the SAME reason, and the skip count
// does not turn into a failure that would dilute accounting. (Belt-and-braces
// over the Record contract, scoped to the gating path.)
func TestRunOpSkipDoesNotFeedBreaker(t *testing.T) {
	rec := NewRecorder()
	b := NewBreaker(5, 1.0, 100)
	b.Trip("forced")
	r := &Run{Recorder: rec, Breaker: b, Cleanup: NewCleanup()}
	c := fakeCycle{"readonly", KindRead}

	for i := 0; i < 10; i++ {
		_ = r.op(c, "x.list", func() error { return client.StatusError{Code: 500} })
	}
	if b.Reason() != "forced" {
		t.Fatalf("gated skips must not re-feed the breaker; reason changed to %q", b.Reason())
	}
	for _, o := range rec.Ops() {
		if !o.Skipped {
			t.Fatalf("every op after a trip must be a skip, got %+v", o)
		}
	}
}

// TestRunOpBreakerTripsOnDistressNot4xx pins the breaker-distress fix: op()
// feeds the breaker cat.isDistress(), NOT cat.isFailure(). A burst of
// deterministic 4xx (client errors) must NOT trip the breaker — they are real
// failures in the report but not API distress, and tripping would mask the rest
// of the map. A 429 (rate-limited) and a 5xx MUST trip.
//
// Mutation proof: revert Run.op to feed cat.isFailure() and the 4xx burst trips
// the breaker, so the "must NOT trip" assertion goes RED.
func TestRunOpBreakerTripsOnDistressNot4xx(t *testing.T) {
	c := fakeCycle{"readonly", KindRead}

	t.Run("4xx burst does NOT trip", func(t *testing.T) {
		b := NewBreaker(3, 1.0, 100)
		r := &Run{Recorder: NewRecorder(), Breaker: b, Cleanup: NewCleanup()}
		for i := 0; i < 10; i++ {
			_ = r.op(c, "x.list", func() error { return client.StatusError{Code: 403} })
		}
		if b.Tripped() {
			t.Fatal("a burst of deterministic 4xx must NOT trip the breaker (not API distress)")
		}
	})

	t.Run("429 burst trips (rate limiting is distress)", func(t *testing.T) {
		b := NewBreaker(3, 1.0, 100)
		r := &Run{Recorder: NewRecorder(), Breaker: b, Cleanup: NewCleanup()}
		for i := 0; i < 3; i++ {
			_ = r.op(c, "x.list", func() error { return client.StatusError{Code: 429} })
		}
		if !b.Tripped() {
			t.Fatal("429 (rate limiting) IS distress and must trip the breaker")
		}
	})

	t.Run("5xx burst trips", func(t *testing.T) {
		b := NewBreaker(3, 1.0, 100)
		r := &Run{Recorder: NewRecorder(), Breaker: b, Cleanup: NewCleanup()}
		for i := 0; i < 3; i++ {
			_ = r.op(c, "x.list", func() error { return client.StatusError{Code: 503} })
		}
		if !b.Tripped() {
			t.Fatal("a 5xx burst must trip the breaker (real distress)")
		}
	})
}
