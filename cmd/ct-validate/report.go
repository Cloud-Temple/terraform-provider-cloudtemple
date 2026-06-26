package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
	"time"
)

// squeakThreshold is the success-rate ceiling below which an endpoint is
// flagged in the "WHERE IT SQUEAKS" summary. Anything at or below this is
// considered a hot spot worth an operator's eyes.
const squeakThreshold = 1.0

// jsonReport is the machine-readable shape emitted with -json.
type jsonReport struct {
	Selected   []string              `json:"selected_cycles"`
	GatedWrite []string              `json:"gated_write_cycles,omitempty"`
	Tripped    bool                  `json:"circuit_tripped"`
	TripReason string                `json:"trip_reason,omitempty"`
	Squeaks    []squeak              `json:"where_it_squeaks"`
	Endpoints  []jsonEndpoint        `json:"endpoints"`
	Teardown   []jsonTeardownFailure `json:"teardown_failures,omitempty"`
	HasFailure bool                  `json:"has_failure"`
}

type squeak struct {
	Cycle       string  `json:"cycle"`
	Endpoint    string  `json:"endpoint"`
	SuccessRate float64 `json:"success_rate"`
	Total       int     `json:"total"`
	SampleError string  `json:"sample_error,omitempty"`
}

type jsonEndpoint struct {
	Cycle       string           `json:"cycle"`
	Endpoint    string           `json:"endpoint"`
	Total       int              `json:"total"`
	OK          int              `json:"ok"`
	Skipped     int              `json:"skipped"`
	SuccessRate float64          `json:"success_rate"`
	P50ms       float64          `json:"p50_ms"`
	P95ms       float64          `json:"p95_ms"`
	Reasons     map[Category]int `json:"reasons"`
	SampleError string           `json:"sample_error,omitempty"`
}

type jsonTeardownFailure struct {
	Label    string `json:"label"`
	Attempts int    `json:"attempts"`
	Error    string `json:"error"`
}

// HasFailure reports whether any recorded endpoint had at least one failure
// (skips and successes excluded). It is the basis for the process exit code,
// so the tool is CI/automation friendly.
func HasFailure(stats []EndpointStats) bool {
	for _, s := range stats {
		attempted := s.Total - s.Skipped
		if attempted > 0 && s.OK < attempted {
			return true
		}
	}
	return false
}

// squeaks returns the endpoints with the worst success rate (rate <
// squeakThreshold), worst first, then by cycle/endpoint for stability.
func squeaks(stats []EndpointStats) []squeak {
	var out []squeak
	for _, s := range stats {
		rate := s.SuccessRate()
		if rate < squeakThreshold {
			out = append(out, squeak{
				Cycle:       s.Cycle,
				Endpoint:    s.Endpoint,
				SuccessRate: rate,
				Total:       s.Total,
				SampleError: s.SampleError,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SuccessRate != out[j].SuccessRate {
			return out[i].SuccessRate < out[j].SuccessRate
		}
		if out[i].Cycle != out[j].Cycle {
			return out[i].Cycle < out[j].Cycle
		}
		return out[i].Endpoint < out[j].Endpoint
	})
	return out
}

// topReasons returns up to n failure categories (excluding OK and Skipped),
// most frequent first, formatted as "category=count".
func topReasons(reasons map[Category]int, n int) string {
	type kv struct {
		cat Category
		cnt int
	}
	var kvs []kv
	for cat, cnt := range reasons {
		if cat == CategoryOK || cat == CategorySkipped {
			continue
		}
		kvs = append(kvs, kv{cat, cnt})
	}
	sort.Slice(kvs, func(i, j int) bool {
		if kvs[i].cnt != kvs[j].cnt {
			return kvs[i].cnt > kvs[j].cnt
		}
		return kvs[i].cat < kvs[j].cat
	})
	if len(kvs) == 0 {
		return "-"
	}
	out := ""
	for i, e := range kvs {
		if i >= n {
			break
		}
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%s=%d", e.cat, e.cnt)
	}
	return out
}

func msFloat(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// PrintText renders the human-readable report to w.
func PrintText(w io.Writer, res EngineResult) {
	fmt.Fprintln(w, "==================== ct-validate report ====================")
	fmt.Fprintf(w, "Cycles run: %v\n", res.SelectedCycles)
	if len(res.GatedWriteSkips) > 0 {
		fmt.Fprintf(w, "Write cycles GATED (no -write): %v\n", res.GatedWriteSkips)
	}
	if res.Tripped {
		fmt.Fprintf(w, "CIRCUIT BREAKER TRIPPED: %s\n", res.TripReason)
		fmt.Fprintln(w, "  -> the engine stopped launching new work and tore down. Treat the API as in distress.")
	}
	fmt.Fprintln(w)

	sq := squeaks(res.Stats)
	fmt.Fprintln(w, "---- WHERE IT SQUEAKS (worst success rate first) ----")
	if len(sq) == 0 {
		fmt.Fprintln(w, "  (nothing squeaked: every attempted endpoint succeeded)")
	} else {
		for _, s := range sq {
			fmt.Fprintf(w, "  %-6.1f%%  %s / %s  (n=%d)\n", s.SuccessRate*100, s.Cycle, s.Endpoint, s.Total)
			if s.SampleError != "" {
				fmt.Fprintf(w, "           ↳ %s\n", s.SampleError)
			}
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "---- per-endpoint ----")
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "CYCLE\tENDPOINT\tRUNS\tOK%\tP50(ms)\tP95(ms)\tTOP FAILURES")
	for _, s := range res.Stats {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%.0f\t%.1f\t%.1f\t%s\n",
			s.Cycle, s.Endpoint, s.Total, s.SuccessRate()*100,
			msFloat(s.P50), msFloat(s.P95), topReasons(s.Reasons, 3))
	}
	tw.Flush()

	if len(res.TeardownFailed) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "---- TEARDOWN FAILURES (possible orphans — investigate) ----")
		for _, f := range res.TeardownFailed {
			fmt.Fprintf(w, "  %s (after %d attempt(s)): %v\n", f.Label, f.Attempts, f.Err)
		}
	}

	fmt.Fprintln(w)
	if HasFailure(res.Stats) {
		fmt.Fprintln(w, "RESULT: FAILURES PRESENT (exit non-zero)")
	} else {
		fmt.Fprintln(w, "RESULT: all attempted endpoints succeeded")
	}
}

// PrintJSON renders the machine-readable report to w.
func PrintJSON(w io.Writer, res EngineResult) error {
	rep := jsonReport{
		Selected:   res.SelectedCycles,
		GatedWrite: res.GatedWriteSkips,
		Tripped:    res.Tripped,
		TripReason: res.TripReason,
		Squeaks:    squeaks(res.Stats),
		HasFailure: HasFailure(res.Stats),
	}
	for _, s := range res.Stats {
		rep.Endpoints = append(rep.Endpoints, jsonEndpoint{
			Cycle:       s.Cycle,
			Endpoint:    s.Endpoint,
			Total:       s.Total,
			OK:          s.OK,
			Skipped:     s.Skipped,
			SuccessRate: s.SuccessRate(),
			P50ms:       msFloat(s.P50),
			P95ms:       msFloat(s.P95),
			Reasons:     s.Reasons,
			SampleError: s.SampleError,
		})
	}
	for _, f := range res.TeardownFailed {
		msg := ""
		if f.Err != nil {
			msg = f.Err.Error()
		}
		rep.Teardown = append(rep.Teardown, jsonTeardownFailure{
			Label:    f.Label,
			Attempts: f.Attempts,
			Error:    msg,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}
