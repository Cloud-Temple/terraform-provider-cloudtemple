package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// Kind distinguishes read-only cycles (always safe) from write cycles (gated
// behind -write).
type Kind int

const (
	// KindRead is a read-only cycle: it never mutates and never needs cleanup.
	KindRead Kind = iota
	// KindWrite is a mutating cycle: skipped unless -write is set, and it MUST
	// register teardown for everything it creates.
	KindWrite
)

func (k Kind) String() string {
	if k == KindWrite {
		return "write"
	}
	return "read"
}

// Run carries the per-execution collaborators handed to every cycle: the op
// recorder, the circuit breaker, and the cleanup tracker. A cycle records each
// endpoint call, never retries forever, and registers teardown before a
// created resource can be lost.
type Run struct {
	Recorder *Recorder
	Breaker  *Breaker
	Cleanup  *Cleanup
	// Iteration is the 0-based run index, used to make synthetic identifiers
	// (e.g. MAC addresses) unique across iterations.
	Iteration int
	// Worker is the worker-pool slot index, used together with Iteration to
	// keep concurrent synthetic identifiers unique.
	Worker int
}

// Cycle is a named business cycle exercised against the client. Read cycles run
// regardless of -write; write cycles run only when -write is set.
type Cycle interface {
	Name() string
	Kind() Kind
	// Run executes the cycle once. It records each op via r.Recorder and feeds
	// each outcome to r.Breaker. It returns an error only for a cycle-level
	// abort; per-op failures are recorded, not returned.
	Run(ctx context.Context, c *client.Client, r *Run) error
}

// op times fn, records the outcome (cycle/endpoint/latency/category) on the
// recorder, and feeds the failure signal to the breaker. It returns fn's error
// so the cycle can decide whether to continue. This is the single choke point
// that keeps recording and breaker accounting consistent for every endpoint.
//
// SAFETY (mid-cycle gating): the breaker is consulted BEFORE launching fn. Once
// the breaker has tripped, op does NOT call fn — it records the endpoint as a
// skip (not a failure, so it does not feed the breaker window) and returns nil.
// This bounds the hammering even inside a long, multi-op cycle (e.g. readonly,
// which chains IAM/VPC/Compute/Backup/ObjectStorage/Marketplace/Tag/Activity):
// every post-trip op becomes a cheap no-op instead of another call against a
// distressed shared API. Without this gate a cycle that has already started
// would keep calling every remaining endpoint after an early 502.
func (r *Run) op(c Cycle, endpoint string, fn func() error) error {
	if r.Breaker != nil && !r.Breaker.Allow() {
		r.skip(c, endpoint)
		return nil
	}
	start := time.Now()
	err := fn()
	latency := time.Since(start)
	cat := categorize(err)
	r.Recorder.Record(Op{
		Cycle:    c.Name(),
		Endpoint: endpoint,
		OK:       cat == CategoryOK,
		Latency:  latency,
		Category: cat,
	})
	r.Breaker.Record(cat.isFailure())
	return err
}

// skip records an endpoint as deliberately skipped (no attempt made). It never
// touches the breaker: a skip is not a failure.
func (r *Run) skip(c Cycle, endpoint string) {
	r.Recorder.Record(Op{
		Cycle:    c.Name(),
		Endpoint: endpoint,
		OK:       false,
		Skipped:  true,
		Category: CategorySkipped,
	})
}

// Registry maps cycle name → Cycle.
type Registry struct {
	cycles map[string]Cycle
}

// NewRegistry builds a registry from the given cycles. A duplicate name panics
// (programmer error, caught by the registry unit test).
func NewRegistry(cycles ...Cycle) *Registry {
	r := &Registry{cycles: map[string]Cycle{}}
	for _, c := range cycles {
		if _, dup := r.cycles[c.Name()]; dup {
			panic(fmt.Sprintf("duplicate cycle name %q", c.Name()))
		}
		r.cycles[c.Name()] = c
	}
	return r
}

// Names returns all registered cycle names, sorted.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.cycles))
	for name := range r.cycles {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// All returns all cycles, ordered by name.
func (r *Registry) All() []Cycle {
	names := r.Names()
	out := make([]Cycle, 0, len(names))
	for _, n := range names {
		out = append(out, r.cycles[n])
	}
	return out
}

// Select resolves a CSV cycle spec into the ordered, de-duplicated list of
// cycles to run, applying write-gating.
//
// Rules (all covered by mutation-proven unit tests):
//   - spec is comma-separated; surrounding whitespace and empty tokens are
//     ignored; an all-empty spec is an error.
//   - the special token "all" expands to every registered cycle (ordered by
//     name); "all" combined with other tokens is still just "all".
//   - an unknown name is a hard error (no silent skip).
//   - duplicates collapse to a single entry, preserving first-seen order.
//   - when write is false, write-kind cycles are dropped from the result and
//     reported via the returned `skipped` slice (so the operator sees that a
//     selected write cycle was gated, rather than it vanishing silently).
func (r *Registry) Select(spec string, write bool) (selected []Cycle, skipped []string, err error) {
	tokens := strings.Split(spec, ",")
	var names []string
	useAll := false
	seen := map[string]bool{}
	for _, tok := range tokens {
		name := strings.TrimSpace(tok)
		if name == "" {
			continue
		}
		if name == "all" {
			useAll = true
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}

	if useAll {
		// "all" supersedes any explicit list.
		names = r.Names()
	} else if len(names) == 0 {
		return nil, nil, fmt.Errorf("no cycles selected: spec %q is empty", spec)
	}

	for _, name := range names {
		c, ok := r.cycles[name]
		if !ok {
			return nil, nil, fmt.Errorf("unknown cycle %q (available: %s)", name, strings.Join(r.Names(), ", "))
		}
		if c.Kind() == KindWrite && !write {
			skipped = append(skipped, name)
			continue
		}
		selected = append(selected, c)
	}
	return selected, skipped, nil
}
