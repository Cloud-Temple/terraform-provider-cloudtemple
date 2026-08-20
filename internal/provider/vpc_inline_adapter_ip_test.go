package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// rawAdapters builds a d.GetRawConfig()-shaped value carrying only what these
// helpers read: the os_network_adapter list, each element with network_id and
// ip_address. Using cty directly (rather than a full ResourceData) is what lets
// the tests express the cases the schema cannot: a NULL ip_address versus an
// explicitly empty one, and an UNKNOWN value.
func rawAdapters(t *testing.T, entries []map[string]cty.Value) cty.Value {
	t.Helper()
	if entries == nil {
		return cty.ObjectVal(map[string]cty.Value{
			"os_network_adapter": cty.NullVal(cty.List(cty.Object(map[string]cty.Type{
				"network_id": cty.String, "ip_address": cty.String,
			}))),
		})
	}
	vals := make([]cty.Value, 0, len(entries))
	for _, e := range entries {
		obj := map[string]cty.Value{
			"network_id": cty.NullVal(cty.String),
			"ip_address": cty.NullVal(cty.String),
		}
		for k, v := range e {
			obj[k] = v
		}
		vals = append(vals, cty.ObjectVal(obj))
	}
	return cty.ObjectVal(map[string]cty.Value{"os_network_adapter": cty.TupleVal(vals)})
}

// TestOsAdapterIPConfigured pins that only an EXPLICIT, non-empty ip_address is
// reported as configured, keyed by block index. Reading the merged d.Get map
// instead of the raw config would break every one of these cases.
func TestOsAdapterIPConfigured(t *testing.T) {
	t.Run("a null block list yields nothing", func(t *testing.T) {
		if got := osAdapterIPConfigured(rawAdapters(t, nil)); len(got) != 0 {
			t.Fatalf("want empty, got %v", got)
		}
	})

	t.Run("a null raw config yields nothing", func(t *testing.T) {
		if got := osAdapterIPConfigured(cty.NullVal(cty.EmptyObject)); len(got) != 0 {
			t.Fatalf("want empty, got %v", got)
		}
	})

	t.Run("an unset ip_address is not configured", func(t *testing.T) {
		raw := rawAdapters(t, []map[string]cty.Value{
			{"network_id": cty.StringVal("net-1")},
		})
		if got := osAdapterIPConfigured(raw); len(got) != 0 {
			t.Fatalf("an absent ip_address must not be reported as configured, got %v", got)
		}
	})

	t.Run("an explicitly EMPTY ip_address is not configured", func(t *testing.T) {
		raw := rawAdapters(t, []map[string]cty.Value{
			{"network_id": cty.StringVal("net-1"), "ip_address": cty.StringVal("")},
		})
		if got := osAdapterIPConfigured(raw); len(got) != 0 {
			t.Fatalf("an empty ip_address carries no address to push, got %v", got)
		}
	})

	t.Run("an UNKNOWN ip_address is not configured (fail-safe)", func(t *testing.T) {
		raw := rawAdapters(t, []map[string]cty.Value{
			{"network_id": cty.StringVal("net-1"), "ip_address": cty.UnknownVal(cty.String)},
		})
		if got := osAdapterIPConfigured(raw); len(got) != 0 {
			t.Fatalf("an unknown ip_address has no concrete value to push, got %v", got)
		}
	})

	t.Run("explicit values are keyed by INDEX, skipping unset ones", func(t *testing.T) {
		raw := rawAdapters(t, []map[string]cty.Value{
			{"network_id": cty.StringVal("net-0")},
			{"network_id": cty.StringVal("net-1"), "ip_address": cty.StringVal("10.0.1.5")},
			{"network_id": cty.StringVal("net-2"), "ip_address": cty.StringVal("10.0.2.6")},
		})
		got := osAdapterIPConfigured(raw)
		if len(got) != 2 || got[1] != "10.0.1.5" || got[2] != "10.0.2.6" {
			t.Fatalf("index keying wrong: %v", got)
		}
		if _, ok := got[0]; ok {
			t.Fatalf("index 0 sets no ip_address and must be absent: %v", got)
		}
	})
}

// TestValidateInlineAdapterIPsTargetVPC pins the fail-closed precondition: an
// ip_address is only valid on an existing, VPC-backed network, and the verdict is
// reached WITHOUT any side effect.
func TestValidateInlineAdapterIPsTargetVPC(t *testing.T) {
	ctx := context.Background()
	netAt := func(m map[int]string) func(int) string {
		return func(i int) string { return m[i] }
	}

	t.Run("no configured ip_address reads no network at all", func(t *testing.T) {
		called := 0
		status := func(ctx context.Context, id string) (bool, bool, error) {
			called++
			return true, true, nil
		}
		if diags := validateInlineAdapterIPsTargetVPC(ctx, map[int]string{}, netAt(nil), status); diags != nil {
			t.Fatalf("want no diagnostics, got %v", diags)
		}
		if called != 0 {
			t.Fatalf("the network must not be read when no ip_address is configured (got %d reads)", called)
		}
	})

	t.Run("a VPC-backed network passes", func(t *testing.T) {
		status := func(ctx context.Context, id string) (bool, bool, error) { return true, true, nil }
		diags := validateInlineAdapterIPsTargetVPC(ctx, map[int]string{0: "10.0.0.5"}, netAt(map[int]string{0: "net-vpc"}), status)
		if diags != nil {
			t.Fatalf("a VPC network must pass: %v", diags)
		}
	})

	t.Run("a non-VPC network is refused", func(t *testing.T) {
		status := func(ctx context.Context, id string) (bool, bool, error) { return false, true, nil }
		diags := validateInlineAdapterIPsTargetVPC(ctx, map[int]string{0: "10.0.0.5"}, netAt(map[int]string{0: "net-pb"}), status)
		if diags == nil {
			t.Fatal("a non-VPC network must be refused")
		}
		if !strings.Contains(diags[0].Summary, "ip_address") {
			t.Fatalf("the diagnostic must name ip_address: %q", diags[0].Summary)
		}
	})

	t.Run("an absent network is refused", func(t *testing.T) {
		status := func(ctx context.Context, id string) (bool, bool, error) { return false, false, nil }
		if diags := validateInlineAdapterIPsTargetVPC(ctx, map[int]string{0: "10.0.0.5"}, netAt(map[int]string{0: "gone"}), status); diags == nil {
			t.Fatal("an absent network must be refused")
		}
	})

	t.Run("an unreadable network fails CLOSED", func(t *testing.T) {
		status := func(ctx context.Context, id string) (bool, bool, error) { return false, false, errors.New("403") }
		diags := validateInlineAdapterIPsTargetVPC(ctx, map[int]string{0: "10.0.0.5"}, netAt(map[int]string{0: "net-x"}), status)
		if diags == nil {
			t.Fatal("an unreadable network must fail closed, never be assumed VPC-backed")
		}
	})

	t.Run("a block with no network_id is refused rather than pushed blindly", func(t *testing.T) {
		called := 0
		status := func(ctx context.Context, id string) (bool, bool, error) { called++; return true, true, nil }
		diags := validateInlineAdapterIPsTargetVPC(ctx, map[int]string{0: "10.0.0.5"}, netAt(nil), status)
		if diags == nil {
			t.Fatal("an ip_address with no declared network_id must be refused")
		}
		if called != 0 {
			t.Fatalf("no network read is possible without a network_id (got %d)", called)
		}
	})
}

// TestRejectInlineAdapterIPSharedNetwork pins the constraint MEASURED live: the
// platform assigns an explicit address per (VM, network) pair, so it refuses one
// address when several adapters sit on that network. Catching it here turns a
// post-create platform failure into a pre-flight refusal.
func TestRejectInlineAdapterIPSharedNetwork(t *testing.T) {
	netAt := func(m map[int]string) func(int) string {
		return func(i int) string { return m[i] }
	}

	t.Run("two blocks on the SAME network with one addressed is refused", func(t *testing.T) {
		diags := rejectInlineAdapterIPSharedNetwork(
			map[int]string{0: "10.0.0.238"},
			netAt(map[int]string{0: "net-vpc", 1: "net-vpc"}),
			2,
		)
		if diags == nil {
			t.Fatal("the platform refuses this; the provider must refuse it first")
		}
		if !strings.Contains(diags[0].Summary, "same network") {
			t.Fatalf("the diagnostic must explain the collision: %q", diags[0].Summary)
		}
	})

	t.Run("two blocks on DIFFERENT networks pass", func(t *testing.T) {
		diags := rejectInlineAdapterIPSharedNetwork(
			map[int]string{0: "10.0.0.238"},
			netAt(map[int]string{0: "net-vpc-a", 1: "net-vpc-b"}),
			2,
		)
		if diags != nil {
			t.Fatalf("distinct networks are legitimate: %v", diags)
		}
	})

	t.Run("a single addressed block passes", func(t *testing.T) {
		if diags := rejectInlineAdapterIPSharedNetwork(map[int]string{0: "10.0.0.238"}, netAt(map[int]string{0: "net-vpc"}), 1); diags != nil {
			t.Fatalf("want no diagnostics: %v", diags)
		}
	})

	t.Run("no configured address means no collision to look for", func(t *testing.T) {
		if diags := rejectInlineAdapterIPSharedNetwork(map[int]string{}, netAt(map[int]string{0: "net-vpc", 1: "net-vpc"}), 2); diags != nil {
			t.Fatalf("blocks sharing a network without an address are legitimate: %v", diags)
		}
	})

	t.Run("a block with no resolvable network is left to the other checks", func(t *testing.T) {
		if diags := rejectInlineAdapterIPSharedNetwork(map[int]string{0: "10.0.0.238"}, netAt(nil), 2); diags != nil {
			t.Fatalf("an unresolvable network is validateInlineAdapterIPsTargetVPC's business: %v", diags)
		}
	})
}

// TestRejectInlineAdapterIPWithoutLiveAdapter pins that a configured static IP is
// never silently discarded by the live-sized state write, while an over-declared
// block that does NOT set ip_address keeps its long-standing (tolerated) behaviour.
func TestRejectInlineAdapterIPWithoutLiveAdapter(t *testing.T) {
	t.Run("an ip_address within the live adapter count passes", func(t *testing.T) {
		if diags := rejectInlineAdapterIPWithoutLiveAdapter(map[int]string{0: "10.0.0.5"}, 1, "vm-1"); diags != nil {
			t.Fatalf("want no diagnostics: %v", diags)
		}
	})

	t.Run("an ip_address beyond the live adapter count is refused", func(t *testing.T) {
		diags := rejectInlineAdapterIPWithoutLiveAdapter(map[int]string{1: "10.0.0.6"}, 1, "vm-1")
		if diags == nil {
			t.Fatal("a static IP on a block with no live adapter must be refused, not silently dropped")
		}
		if !strings.Contains(diags[0].Summary, "silently dropped") {
			t.Fatalf("the diagnostic must explain WHY: %q", diags[0].Summary)
		}
	})

	t.Run("no configured ip_address means no rejection, whatever the counts", func(t *testing.T) {
		if diags := rejectInlineAdapterIPWithoutLiveAdapter(map[int]string{}, 0, "vm-1"); diags != nil {
			t.Fatalf("over-declared blocks without ip_address keep their existing behaviour: %v", diags)
		}
	})
}

// TestPreserveInlineAdapterIP pins the read-path carry-across. Without it the
// write-only ip_address is blanked on every refresh and the user gets a permanent
// diff — the failure this helper exists to prevent.
func TestPreserveInlineAdapterIP(t *testing.T) {
	t.Run("a previous ip_address survives a live flatten that has none", func(t *testing.T) {
		prev := map[string]interface{}{"id": "nic-1", "ip_address": "10.0.6.240"}
		flat := map[string]interface{}{"id": "nic-1", "network_id": "net-vpc"}
		got := preserveInlineAdapterIP(prev, flat).(map[string]interface{})
		if got["ip_address"] != "10.0.6.240" {
			t.Fatalf("ip_address = %v, want it carried across from the previous state", got["ip_address"])
		}
		if got["network_id"] != "net-vpc" {
			t.Fatalf("the live values must be untouched, got %v", got)
		}
	})

	t.Run("an empty previous value writes nothing", func(t *testing.T) {
		prev := map[string]interface{}{"id": "nic-1", "ip_address": ""}
		flat := map[string]interface{}{"id": "nic-1"}
		got := preserveInlineAdapterIP(prev, flat).(map[string]interface{})
		if v, ok := got["ip_address"]; ok && v != "" {
			t.Fatalf("an empty previous value must not be written, got %v", v)
		}
	})

	t.Run("a non-map previous entry is tolerated", func(t *testing.T) {
		flat := map[string]interface{}{"id": "nic-1"}
		if got := preserveInlineAdapterIP(nil, flat).(map[string]interface{}); got["id"] != "nic-1" {
			t.Fatalf("a nil previous entry must leave the flatten intact, got %v", got)
		}
		// A flatten that is not a map at all must pass through, never panic.
		if got := preserveInlineAdapterIP(map[string]interface{}{"ip_address": "10.0.0.1"}, "not-a-map"); got != "not-a-map" {
			t.Fatalf("a non-map flatten must pass through untouched, got %v", got)
		}
	})
}

// TestSortedIndexes and the multi-offender cases below pin that a diagnostic
// always names the LOWEST offending block. Go randomises map iteration, so a
// check returning on its first entry would name a different block from run to run
// — unreproducible for the user and flaky for the test asserting on it.
func TestSortedIndexes(t *testing.T) {
	got := sortedIndexes(map[int]string{3: "c", 0: "a", 2: "b"})
	if len(got) != 3 || got[0] != 0 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("want ascending [0 2 3], got %v", got)
	}
	if len(sortedIndexes(map[int]string{})) != 0 {
		t.Fatal("an empty map must yield no indexes")
	}
}

func TestInlineAdapterIPDiagnosticsAreDeterministic(t *testing.T) {
	// Two offenders: the reported index must be 1 (the lowest), every time.
	// Repeated because a map-order bug is probabilistic and would otherwise slip
	// through a single run.
	netAt := func(i int) string { return map[int]string{1: "net-pb", 2: "net-pb2"}[i] }
	status := func(ctx context.Context, id string) (bool, bool, error) { return false, true, nil }
	for i := 0; i < 50; i++ {
		diags := validateInlineAdapterIPsTargetVPC(context.Background(),
			map[int]string{1: "10.0.0.1", 2: "10.0.0.2"}, netAt, status)
		if diags == nil {
			t.Fatal("both blocks are invalid; one must be reported")
		}
		if !strings.Contains(diags[0].Summary, "10.0.0.1") {
			t.Fatalf("run %d reported %q, want the LOWEST offending block (10.0.0.1)", i, diags[0].Summary)
		}
	}

	for i := 0; i < 50; i++ {
		diags := rejectInlineAdapterIPWithoutLiveAdapter(map[int]string{2: "10.0.0.2", 5: "10.0.0.5"}, 1, "vm-1")
		if diags == nil {
			t.Fatal("both blocks are beyond the live count; one must be reported")
		}
		if !strings.Contains(diags[0].Summary, "os_network_adapter[2]") {
			t.Fatalf("run %d reported %q, want the LOWEST offending index (2)", i, diags[0].Summary)
		}
	}
}

// TestRefuseBeforeAnyMutation pins the state-safety property directly on a
// ResourceData carrying a PRIOR state plus PLANNED changes: after a refusal, the
// state that would be persisted must be the PRIOR one, not the planned one.
//
// This is not a test of the SDK; it is a test of the invariant this provider
// relies on. If a future SDK changed the meaning of Partial, a refused apply would
// silently start recording rejected configuration again — a write-only attribute
// like ip_address is never corrected by a refresh, so the false intent would stick.
func TestRefuseBeforeAnyMutation(t *testing.T) {
	res := resourceOpenIaasVirtualMachine()

	priorState := terraform.NewInstanceStateShimmedFromValue(cty.ObjectVal(map[string]cty.Value{
		"id":   cty.StringVal("vm-1"),
		"name": cty.StringVal("before"),
	}), res.SchemaVersion)
	priorState.ID = "vm-1"

	newData := func(t *testing.T) *schema.ResourceData {
		t.Helper()
		d, err := res.Data(priorState), error(nil)
		if err != nil {
			t.Fatalf("Data: %v", err)
		}
		if err := d.Set("name", "after"); err != nil {
			t.Fatalf("Set(name): %v", err)
		}
		return d
	}

	t.Run("nil diagnostics change nothing", func(t *testing.T) {
		d := newData(t)
		if got := refuseBeforeAnyMutation(d, nil); got != nil {
			t.Fatalf("want nil diagnostics, got %v", got)
		}
		if got := d.State().Attributes["name"]; got != "after" {
			t.Fatalf("without a refusal the planned value must survive, got %q", got)
		}
	})

	t.Run("a refusal preserves the PRIOR state", func(t *testing.T) {
		d := newData(t)
		in := diag.Errorf("refused")
		out := refuseBeforeAnyMutation(d, in)
		if out == nil || !out.HasError() {
			t.Fatal("the diagnostics must be passed through unchanged")
		}
		if got := d.State().Attributes["name"]; got != "before" {
			t.Fatalf("name = %q, want the PRIOR value %q — a refused apply must not record planned values", got, "before")
		}
	})
}

// TestInlineAdaptersNeedCollection pins the REACHABILITY half of the ip_address
// convergence contract. The decision half (the patch itself) is pinned elsewhere;
// this is the half that is easy to lose, because the natural gate — "did the
// os_network_adapter block change?" — is exactly the wrong one for a write-only
// attribute whose stored value the read path preserves.
//
// The failure it forbids: a refusal late in an apply persists the planned
// ip_address, the read preserves it, so state == config, HasChange is false, the
// reconciliation is never entered and the address is NEVER applied. Permanent, not
// transient. Narrowing this predicate back to `adaptersChanged` reopens it.
func TestInlineAdaptersNeedCollection(t *testing.T) {
	cases := []struct {
		name       string
		changed    bool
		configured map[int]string
		want       bool
	}{
		{"an adapter diff collects, as before", true, nil, true},
		{"no diff and no configured address collects nothing", false, map[int]string{}, false},
		{"NO DIFF but a configured address MUST still collect", false, map[int]string{0: "10.0.0.5"}, true},
		{"both reasons at once still collects", true, map[int]string{1: "10.0.0.6"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inlineAdaptersNeedCollection(tc.changed, tc.configured); got != tc.want {
				t.Fatalf("inlineAdaptersNeedCollection(%v, %v) = %v, want %v", tc.changed, tc.configured, got, tc.want)
			}
		})
	}
}

// TestWithPriorInlineAdapterIPs pins BOTH properties of the failure-path rollback:
// how narrow it is per field, and how WIDE it is across adapters.
//
// Narrow per field, because a failed VPC relocation may follow a network or MAC patch
// of the same apply that DID succeed — erasing those would destroy a real change.
// Wide across adapters, because the reconciliation loop aborts on the first failure,
// so a later addressed adapter was never attempted and its planned address must not
// be recorded. Over-reverting costs one no-op apply; under-reverting is permanent:
// an unattempted address left in the state kills the diff and Update never returns.
func TestWithPriorInlineAdapterIPs(t *testing.T) {
	prior := []interface{}{
		map[string]interface{}{"id": "vif-1", "network_id": "net-a", "ip_address": "10.0.0.1", "mac_address": "aa:aa"},
		map[string]interface{}{"id": "vif-2", "network_id": "net-b", "ip_address": "10.0.0.2"},
	}
	planned := []interface{}{
		// network_id and mac_address changed AND were applied; only the address failed.
		map[string]interface{}{"id": "vif-1", "network_id": "net-c", "ip_address": "10.0.0.9", "mac_address": "bb:bb"},
		// never attempted, because the loop aborted on vif-1.
		map[string]interface{}{"id": "vif-2", "network_id": "net-b", "ip_address": "10.0.0.8"},
	}

	got := withPriorInlineAdapterIPs(planned, prior)

	first := got[0].(map[string]interface{})
	if first["ip_address"] != "10.0.0.1" {
		t.Fatalf("vif-1 ip_address = %v, want the PRIOR value: the address was not applied", first["ip_address"])
	}
	if first["network_id"] != "net-c" || first["mac_address"] != "bb:bb" {
		t.Fatalf("vif-1: the APPLIED network/mac changes must survive, got %v", first)
	}

	second := got[1].(map[string]interface{})
	if second["ip_address"] != "10.0.0.2" {
		t.Fatalf("vif-2 ip_address = %v, want the PRIOR value: it was NEVER ATTEMPTED, so leaving the planned address would kill the next diff and never be applied", second["ip_address"])
	}
	if second["network_id"] != "net-b" {
		t.Fatalf("vif-2: non-IP fields must be untouched, got %v", second)
	}

	// The input must not be mutated in place: the planned list is shared with the
	// caller's own view of the plan.
	if planned[0].(map[string]interface{})["ip_address"] != "10.0.0.9" {
		t.Fatal("withPriorInlineAdapterIPs must not mutate its input")
	}
	if planned[1].(map[string]interface{})["ip_address"] != "10.0.0.8" {
		t.Fatal("withPriorInlineAdapterIPs must not mutate its input")
	}

	t.Run("an adapter absent from the prior state has its address cleared", func(t *testing.T) {
		got := withPriorInlineAdapterIPs(planned, []interface{}{})
		for i, e := range got {
			if e.(map[string]interface{})["ip_address"] != "" {
				t.Fatalf("entry %d: with no prior value the address must be cleared, got %v", i, e)
			}
		}
	})

	t.Run("an entry that is not a map passes through", func(t *testing.T) {
		got := withPriorInlineAdapterIPs([]interface{}{nil}, prior)
		if len(got) != 1 || got[0] != nil {
			t.Fatalf("a nil entry must pass through untouched, got %v", got)
		}
	})
}
