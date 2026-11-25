package helpers

import (
	"math"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// policyWithLastRun builds a named policy carrying the given millisecond
// timestamps, so a diagnostic can be checked for the identity it must name.
func policyWithLastRun(start, end int64) *client.OpenIaaSReplicationPolicy {
	p := &client.OpenIaaSReplicationPolicy{ID: "pol-1", Name: "nightly"}
	p.LastRun.Start = start
	p.LastRun.End = end
	p.LastRun.Status = "SUCCESS"
	return p
}

// TestValidateOpenIaaSReplicationPolicyLastRunAcceptsRepresentableTimestamps
// pins the accepted domain. The bound is inclusive: 2^53 is the last integer a
// float64 holds exactly, so refusing it would be a false positive.
func TestValidateOpenIaaSReplicationPolicyLastRunAcceptsRepresentableTimestamps(t *testing.T) {
	for name, p := range map[string]*client.OpenIaaSReplicationPolicy{
		"zero (the API omitted lastRun)": policyWithLastRun(0, 0),
		"a contemporary run":             policyWithLastRun(1759830000000, 1759830012345),
		"exactly +2^53":                  policyWithLastRun(MaxExactTimestampMs, MaxExactTimestampMs),
		"exactly -2^53":                  policyWithLastRun(-MaxExactTimestampMs, -MaxExactTimestampMs),
		"nil policy":                     nil,
	} {
		if err := ValidateOpenIaaSReplicationPolicyLastRun(p); err != nil {
			t.Errorf("%s must be accepted, got: %v", name, err)
		}
	}
}

// TestValidateOpenIaaSReplicationPolicyLastRunRefusesInexactTimestamps is the
// state-safety gate. `last_run` is stored as a float64 (schema.TypeFloat), and
// a float64 rounds every integer beyond 2^53: 9007199254740993 becomes
// 9007199254740992. Recording the rounded value would leave the state silently
// different from the platform, which the doctrine forbids — the read fails
// closed instead, naming the field and the offending value.
//
// Mutation proven RED: dropping either bound from the comparison in
// ValidateOpenIaaSReplicationPolicyLastRun, or widening it to math.MaxInt64,
// turns the matching sub-test red.
func TestValidateOpenIaaSReplicationPolicyLastRunRefusesInexactTimestamps(t *testing.T) {
	// Sanity: the counterexample really is the first value float64 cannot hold,
	// so the bound is derived from the arithmetic rather than asserted.
	const firstInexact = MaxExactTimestampMs + 1
	if int64(float64(firstInexact)) != MaxExactTimestampMs {
		t.Fatalf("precondition broken: float64(%d) should round back to %d", firstInexact, MaxExactTimestampMs)
	}

	for name, tc := range map[string]struct {
		policy *client.OpenIaaSReplicationPolicy
		field  string
		value  string
	}{
		"start just past the exact range": {policyWithLastRun(firstInexact, 0), "start", "9007199254740993"},
		"end just past the exact range":   {policyWithLastRun(0, firstInexact), "end", "9007199254740993"},
		"start far past the exact range":  {policyWithLastRun(math.MaxInt64, 0), "start", "9223372036854775807"},
		"negative past the exact range":   {policyWithLastRun(0, -firstInexact), "end", "-9007199254740993"},
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateOpenIaaSReplicationPolicyLastRun(tc.policy)
			if err == nil {
				t.Fatalf("a timestamp a float64 cannot hold exactly must be refused, got no error")
			}
			msg := err.Error()
			// The diagnostic must be actionable: which policy, which field,
			// which value. Without those an operator cannot act on it.
			for _, want := range []string{"pol-1", "nightly", "last_run." + tc.field, tc.value} {
				if !strings.Contains(msg, want) {
					t.Errorf("the diagnostic must name %q, got: %s", want, msg)
				}
			}
		})
	}
}

// TestFlattenOpenIaaSReplicationPolicyPassesTimestampsThroughAsInt64 pins the
// contract between the client struct and the state writer: the mapping layer
// must hand the raw int64 over, never a narrowed or pre-rounded value. A
// regression to `int` here would be invisible on a 64-bit host at the value
// level, so the dynamic type is what is asserted.
func TestFlattenOpenIaaSReplicationPolicyPassesTimestampsThroughAsInt64(t *testing.T) {
	flat := FlattenOpenIaaSReplicationPolicy(policyWithLastRun(1759830000000, 1759830012345))

	lastRun, ok := flat["last_run"].([]interface{})
	if !ok || len(lastRun) != 1 {
		t.Fatalf("last_run must be a one-element list, got %#v", flat["last_run"])
	}
	entry, ok := lastRun[0].(map[string]interface{})
	if !ok {
		t.Fatalf("last_run[0] must be a map, got %#v", lastRun[0])
	}

	for field, want := range map[string]int64{"start": 1759830000000, "end": 1759830012345} {
		got, ok := entry[field].(int64)
		if !ok {
			t.Fatalf("last_run.%s must be emitted as int64, got %T (%v)", field, entry[field], entry[field])
		}
		if got != want {
			t.Fatalf("last_run.%s = %d, want %d", field, got, want)
		}
	}
}
