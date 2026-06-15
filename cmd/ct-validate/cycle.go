package main

import (
	"context"
	"fmt"
	"regexp"
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
	detail := ""
	if cat != CategoryOK && err != nil {
		// Keep the failure reason (e.g. the 4xx/5xx body) so the report can show
		// WHY it squeaked. Redact obvious secret carriers FIRST (defence-in-depth
		// for a report that may be shared), then bound it so a huge body cannot
		// bloat the recording.
		detail = truncate(redactSecrets(err.Error()), 300)
	}
	r.Recorder.Record(Op{
		Cycle:    c.Name(),
		Endpoint: endpoint,
		OK:       cat == CategoryOK,
		Latency:  latency,
		Category: cat,
		Detail:   detail,
	})
	// The breaker trips on DISTRESS only (5xx, 502, timeout, transient, 429),
	// NOT on deterministic client errors (4xx): a 4xx is recorded as a failure
	// in the report but must not latch the breaker and mask the rest of the map.
	r.Breaker.Record(cat.isDistress())
	return err
}

// secret-bearing patterns scrubbed from recorded error text before it is stored
// or printed. The PAT travels in the Authorization header and is not normally
// echoed in an API response body, but a proxy/debug body could reflect request
// metadata — so mask the obvious carriers rather than trust that it never will.
var (
	// An Authorization value of ANY scheme (Bearer/Basic/Token/ApiKey/…): mask the
	// WHOLE value (scheme AND credential) up to a value delimiter. A scheme alone
	// (e.g. "Basic") must never leave the credential after it exposed.
	authHeaderRe = regexp.MustCompile(`(?i)authorization\s*["']?\s*[:=]\s*["']?[^\r\n"',}&]*`)
	// A bare bearer token not preceded by an Authorization key.
	bearerTokenRe = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/=-]+`)
	// Named secret carriers; the value may be a quoted string (spaces allowed) or a
	// single delimiter-bounded token. Names cover the common OAuth/secret variants.
	kvSecretRe = regexp.MustCompile(`(?i)\b(password|passwd|secret|client_secret|access_token|refresh_token|id_token|token|api[_-]?key|apikey|signature|credential)\b(\s*["']?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s"',}&]+)`)
	// The catch-all: any LONG opaque run (base64/hex/JWT-like, >=20 chars) is
	// credential material regardless of surrounding key/scheme/format — mask it.
	// Ordinary error words are far shorter, so this rarely touches useful text;
	// when in doubt it OVER-redacts, which is the safe direction.
	opaqueTokenRe = regexp.MustCompile(`[A-Za-z0-9+/=_~.-]{20,}`)
)

// redactSecrets scrubs credential material from text that may be recorded or
// printed (an API error body). It layers an Authorization-value mask, a bare
// bearer mask, named secret carriers (quoted or single-token), and finally a
// catch-all long-opaque-token mask — so no credential survives regardless of
// format. It deliberately errs toward OVER-redaction over leaking.
func redactSecrets(s string) string {
	s = authHeaderRe.ReplaceAllString(s, "Authorization: ***REDACTED***")
	s = bearerTokenRe.ReplaceAllString(s, "Bearer ***REDACTED***")
	s = kvSecretRe.ReplaceAllString(s, "${1}=***REDACTED***")
	s = opaqueTokenRe.ReplaceAllString(s, "***REDACTED***")
	return s
}

// truncate collapses newlines (so a recorded error stays one readable line) and
// bounds the result to n runes PLUS a trailing ellipsis when it had to cut. n<=0
// yields an empty string (defensive; call sites pass a positive bound).
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
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
