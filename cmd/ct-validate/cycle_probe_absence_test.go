package main

import (
	"bufio"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestProbeAbsenceOutcome pins the status -> recorded-outcome mapping: 404 is the
// migrated success; 403 and other deterministic non-404 statuses are 4xx failures
// that NEVER trip the breaker; only genuine distress (5xx/429) trips it. A 2xx/3xx
// on a bogus id is clamped to a non-distress 4xx (its real status kept in Detail).
//
// Mutation proof: make 404 return an error, or stop clamping the 2xx/3xx anomaly
// (so it falls to CategoryOther -> distress), or make 403 distress, and a row here
// goes RED.
func TestProbeAbsenceOutcome(t *testing.T) {
	cases := []struct {
		status       int
		wantCat      Category
		wantDistress bool
	}{
		{404, CategoryOK, false},         // migrated
		{403, CategoryHTTP4xx, false},    // NOT migrated (or no permission)
		{409, CategoryHTTP4xx, false},    // other deterministic 4xx
		{200, CategoryHTTP4xx, false},    // anomaly, clamped non-distress
		{301, CategoryHTTP4xx, false},    // anomaly, clamped non-distress
		{500, CategoryHTTP5xx, true},     // distress
		{503, CategoryHTTP5xx, true},     // distress
		{429, CategoryRateLimited, true}, // distress (back off)
	}
	for _, tc := range cases {
		err := probeAbsenceOutcome(tc.status)
		cat := categorize(err)
		if cat != tc.wantCat {
			t.Errorf("status %d: category = %s, want %s", tc.status, cat, tc.wantCat)
		}
		if cat.isDistress() != tc.wantDistress {
			t.Errorf("status %d: distress = %v, want %v", tc.status, cat.isDistress(), tc.wantDistress)
		}
		if tc.status != 404 && !strings.Contains(err.Error(), "absence probe") {
			t.Errorf("status %d: error should mention the absence probe, got %q", tc.status, err.Error())
		}
	}
}

// TestProbeAbsenceCycleRecords drives the cycle against a stub returning a fixed
// status for every endpoint, and asserts the recording and breaker behavior.
func TestProbeAbsenceCycleRecords(t *testing.T) {
	t.Run("all 404 -> every probe OK", func(t *testing.T) {
		c := newReadonlyTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
		_ = probeAbsenceCycle{}.Run(context.Background(), c, r)

		ops := r.Recorder.Ops()
		if len(ops) != len(absenceProbeEndpoints) {
			t.Fatalf("expected %d recorded ops, got %d", len(absenceProbeEndpoints), len(ops))
		}
		for _, o := range ops {
			if !o.OK || o.Category != CategoryOK {
				t.Errorf("%s: a 404 must record OK (migrated), got %+v", o.Endpoint, o)
			}
		}
	})

	t.Run("all 403 -> 4xx not-migrated, breaker never trips", func(t *testing.T) {
		c := newReadonlyTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		b := NewBreaker(1000, 0.99, 1000)
		r := &Run{Recorder: NewRecorder(), Breaker: b, Cleanup: NewCleanup()}
		_ = probeAbsenceCycle{}.Run(context.Background(), c, r)

		for _, o := range r.Recorder.Ops() {
			if o.OK || o.Category != CategoryHTTP4xx {
				t.Errorf("%s: a 403 must record a 4xx not-migrated failure, got %+v", o.Endpoint, o)
			}
			if !strings.Contains(o.Detail, "403") {
				t.Errorf("%s: detail must surface HTTP 403, got %q", o.Endpoint, o.Detail)
			}
		}
		if !b.Allow() {
			t.Fatal("the breaker must NOT trip on deterministic 403s (not distress)")
		}
	})
}

// TestProbeAbsenceRegistryCoverage is the anti-drift guard: it scans the client
// package for every requireNotFoundOrOK(resp, 403) read whose GET path contains a
// %s, and asserts the probe registry covers EXACTLY that set — so a newly added
// by-id read with the old 403 contract cannot ship without a probe entry, and a
// stale entry cannot linger.
func TestProbeAbsenceRegistryCoverage(t *testing.T) {
	expected := scanClient403PathsWithArg(t)

	got := map[string]bool{}
	for _, ep := range absenceProbeEndpoints {
		if got[ep.path] {
			t.Errorf("duplicate registry entry for path %s", ep.path)
		}
		got[ep.path] = true
	}

	for p := range expected {
		if !got[p] {
			t.Errorf("MISSING probe entry: client has a requireNotFoundOrOK(resp,403) read on %q with no absenceProbeEndpoints entry", p)
		}
	}
	for p := range got {
		if !expected[p] {
			t.Errorf("STALE probe entry: absenceProbeEndpoints has %q but no client requireNotFoundOrOK(resp,403) read uses it", p)
		}
	}
}

// scanClient403PathsWithArg returns the set of GET path templates (containing a
// %s) that are read by a method using requireNotFoundOrOK(resp, 403). It mirrors,
// in the test, the inventory the probe registry must match. Within a method the
// GET newRequest always precedes its requireNotFoundOrOK, so tracking the most
// recent GET path correctly attributes it.
func scanClient403PathsWithArg(t *testing.T) map[string]bool {
	t.Helper()
	dir := filepath.Join("..", "..", "internal", "client")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read client dir %s: %v", dir, err)
	}
	getRe := regexp.MustCompile(`newRequest\("GET",\s*"([^"]+)"`)
	nfRe := regexp.MustCompile(`requireNotFoundOrOK\(resp,\s*403\)`)

	out := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		curPath := ""
		for sc.Scan() {
			line := sc.Text()
			if m := getRe.FindStringSubmatch(line); m != nil {
				curPath = m[1]
			}
			if nfRe.MatchString(line) && strings.Contains(curPath, "%s") {
				out[curPath] = true
			}
		}
		_ = f.Close()
	}
	if len(out) == 0 {
		t.Fatal("scan found no requireNotFoundOrOK(resp,403) by-id paths — the scanner or the client layout changed")
	}
	return out
}

// TestProbeAbsenceQuarantinedButRunnable pins that the probe cycle is registered,
// EXCLUDED from the `all` sweep (quarantined: it hits bogus ids), and runnable
// when named explicitly. It is KindRead, so it is not write-gated.
func TestProbeAbsenceQuarantinedButRunnable(t *testing.T) {
	reg := buildRegistry()

	registered := false
	for _, n := range reg.Names() {
		if n == "probe_absence" {
			registered = true
		}
	}
	if !registered {
		t.Fatal("probe_absence must be registered in buildRegistry()")
	}

	all, _, err := reg.Select("all", true)
	if err != nil {
		t.Fatalf("Select(all): %v", err)
	}
	for _, c := range all {
		if c.Name() == "probe_absence" {
			t.Fatal("probe_absence must be EXCLUDED from `all` (quarantined diagnostic)")
		}
	}

	sel, _, err := reg.Select("probe_absence", false)
	if err != nil {
		t.Fatalf("Select(probe_absence): %v", err)
	}
	if len(sel) != 1 || sel[0].Name() != "probe_absence" {
		t.Fatalf("probe_absence must run when named explicitly (KindRead, not write-gated), got %+v", sel)
	}
}
