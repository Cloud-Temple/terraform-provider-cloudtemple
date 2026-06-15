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
		fakeCycle{"vpc", KindWrite},
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
	// With -write, "all" must include the write cycles, ordered by name.
	sel, gated, err := reg.Select("all", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"backup", "object_storage", "readonly", "vpc"}) {
		t.Fatalf("all (write) = %v", got)
	}
	if len(gated) != 0 {
		t.Fatalf("write enabled, nothing should be gated: %v", gated)
	}
}

// TestSelectAllSupersedes proves "all" combined with explicit names is still
// just "all" (no duplicates, full set).
func TestSelectAllSupersedes(t *testing.T) {
	reg := testRegistry()
	sel, _, err := reg.Select("vpc,all,readonly", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"backup", "object_storage", "readonly", "vpc"}) {
		t.Fatalf("all-supersedes = %v", got)
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
// cycles, with the write cycles reported as gated.
func TestSelectAllWriteGating(t *testing.T) {
	reg := testRegistry()
	sel, gated, err := reg.Select("all", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := names(sel); !reflect.DeepEqual(got, []string{"backup", "readonly"}) {
		t.Fatalf("all (read-only) = %v, want [backup readonly]", got)
	}
	if !reflect.DeepEqual(gated, []string{"object_storage", "vpc"}) {
		t.Fatalf("gated = %v, want [object_storage vpc]", gated)
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
