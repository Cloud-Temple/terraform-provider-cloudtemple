package provider

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Issue #543 — 32-bit safety of `last_run.start` / `last_run.end`.
//
// These attributes carry Unix timestamps in MILLISECONDS. schema.TypeInt is
// unusable for them: the SDK writer decodes into a NATIVE int
// (helper/schema/field_writer_map.go, `var n int; mapstructure.Decode(v, &n)`)
// and mapstructure truncates rather than range-checking, so on the 386 and arm
// artifacts goreleaser publishes the state would silently receive a wrong value
// (measured on a 32-bit-sized target: 1759830000000 -> -1106591360). TypeFloat
// decodes into a float64 on every platform.
//
// This file gates the three things that fix has to hold true at once:
//
//  1. the declared type really is TypeFloat, everywhere the block appears;
//  2. the PUBLISHED contract did not move — both TypeInt and TypeFloat map to
//     cty.Number (helper/schema/core_schema.go), which is why the change needs
//     no StateUpgrader; the protocol types below were captured on the schema
//     BEFORE the fix and must still match after it;
//  3. a timestamp written through the real SDK writer comes back out of the
//     state byte-exact, including the `%G` spelling TypeFloat produces.

// replicationLastRunSchemas enumerates every place the block is declared, so a
// new copy of it cannot silently keep TypeInt.
func replicationLastRunSchemas(t *testing.T) map[string]map[string]*schema.Schema {
	t.Helper()

	nested := func(where string, s map[string]*schema.Schema, path ...string) map[string]*schema.Schema {
		for _, key := range path {
			attr, ok := s[key]
			if !ok {
				t.Fatalf("%s: attribute %q is gone", where, key)
			}
			res, ok := attr.Elem.(*schema.Resource)
			if !ok {
				t.Fatalf("%s: attribute %q no longer carries a nested resource", where, key)
			}
			s = res.Schema
		}
		return s
	}

	return map[string]map[string]*schema.Schema{
		"resource cloudtemple_compute_iaas_opensource_replication_policy": nested(
			"resource", resourceOpenIaasReplicationPolicy().Schema, "last_run"),
		"data source cloudtemple_compute_iaas_opensource_replication_policy": nested(
			"data source", dataSourceOpenIaasReplicationPolicy().Schema, "last_run"),
		"data source cloudtemple_compute_iaas_opensource_replication_policies": nested(
			"data sources", dataSourceOpenIaasReplicationPolicies().Schema, "policies", "last_run"),
	}
}

// TestReplicationLastRunTimestampsAreNotStoredAsTypeInt is the RED gate on the
// schema: TypeInt truncates on 32-bit builds. TypeFloat is asserted explicitly
// rather than "anything but TypeInt", because TypeString would also pass such a
// loose check while silently changing the published contract from number to
// string.
func TestReplicationLastRunTimestampsAreNotStoredAsTypeInt(t *testing.T) {
	for where, lastRun := range replicationLastRunSchemas(t) {
		for _, field := range []string{"start", "end"} {
			attr, ok := lastRun[field]
			if !ok {
				t.Errorf("%s: last_run.%s is gone", where, field)
				continue
			}
			if attr.Type == schema.TypeInt {
				t.Errorf("%s: last_run.%s is TypeInt — a millisecond timestamp written through "+
					"the TypeInt writer is truncated to the platform int width, so the 386/arm "+
					"builds would record a wrong value in the state (#543)", where, field)
				continue
			}
			if attr.Type != schema.TypeFloat {
				t.Errorf("%s: last_run.%s is %s, want TypeFloat — any other type moves the "+
					"published cty type away from Number and would need a StateUpgrader",
					where, field, attr.Type)
			}
		}
	}
}

// TestReplicationLastRunProtocolTypesAreUnchanged is the evidence that the
// TypeInt -> TypeFloat switch is NOT a state-shape change and therefore needs no
// StateUpgrader: these strings were captured from `GetProviderSchema` on the
// schema as it stood BEFORE the fix, and are byte-identical after it.
func TestReplicationLastRunProtocolTypesAreUnchanged(t *testing.T) {
	const (
		lastRunType  = `tftypes.List[tftypes.Object["end":tftypes.Number, "start":tftypes.Number, "status":tftypes.String]]`
		policiesType = `tftypes.List[tftypes.Object["id":tftypes.String, "interval":tftypes.List[tftypes.Object["hours":tftypes.Number, "minutes":tftypes.Number]], ` +
			`"last_run":tftypes.List[tftypes.Object["end":tftypes.Number, "start":tftypes.Number, "status":tftypes.String]], ` +
			`"machine_manager_id":tftypes.String, "name":tftypes.String, "pool_id":tftypes.String, "storage_repository_id":tftypes.String]]`
	)

	resp, err := schema.NewGRPCProviderServer(New("543-contract")()).
		GetProviderSchema(context.Background(), &tfprotov5.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}

	attrType := func(s *tfprotov5.Schema, name string) string {
		t.Helper()
		if s == nil {
			t.Fatal("schema is missing from the provider")
		}
		for _, a := range s.Block.Attributes {
			if a.Name == name {
				return a.Type.String()
			}
		}
		t.Fatalf("attribute %q is gone from the schema", name)
		return ""
	}

	for _, tc := range []struct {
		where string
		got   string
		want  string
	}{
		{"resource.last_run",
			attrType(resp.ResourceSchemas["cloudtemple_compute_iaas_opensource_replication_policy"], "last_run"), lastRunType},
		{"data source.last_run",
			attrType(resp.DataSourceSchemas["cloudtemple_compute_iaas_opensource_replication_policy"], "last_run"), lastRunType},
		{"data sources.policies",
			attrType(resp.DataSourceSchemas["cloudtemple_compute_iaas_opensource_replication_policies"], "policies"), policiesType},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: the published protocol type changed (#543 must not move the contract)\n got: %s\nwant: %s",
				tc.where, tc.got, tc.want)
		}
	}
}

// TestReplicationLastRunTimestampsSurviveTheStateWriter runs the real SDK
// writer and reads the stored flatmap back. It is the regression guard on the
// `%G` spelling TypeFloat produces ("1.75983E+12"): the stored value must still
// parse back to the exact millisecond.
func TestReplicationLastRunTimestampsSurviveTheStateWriter(t *testing.T) {
	cases := map[string]struct{ start, end int64 }{
		"zero (the API omitted lastRun)": {0, 0},
		"a contemporary run":             {1759830000000, 1759830012345},
		"the last exactly-storable ms":   {helpers.MaxExactTimestampMs, helpers.MaxExactTimestampMs},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			policy := &client.OpenIaaSReplicationPolicy{ID: "pol-1", Name: "nightly"}
			policy.LastRun.Start = tc.start
			policy.LastRun.End = tc.end
			policy.LastRun.Status = "SUCCESS"

			if err := helpers.ValidateOpenIaaSReplicationPolicyLastRun(policy); err != nil {
				t.Fatalf("the fixture must be inside the representable range: %v", err)
			}
			flat := helpers.FlattenOpenIaaSReplicationPolicy(policy)

			for _, target := range []struct {
				where  string
				res    *schema.Resource
				key    string
				value  interface{}
				prefix string
			}{
				{"resource", resourceOpenIaasReplicationPolicy(), "last_run", flat["last_run"], "last_run.0"},
				{"data source", dataSourceOpenIaasReplicationPolicy(), "last_run", flat["last_run"], "last_run.0"},
				{"data sources", dataSourceOpenIaasReplicationPolicies(), "policies",
					[]map[string]interface{}{flat}, "policies.0.last_run.0"},
			} {
				d := schema.TestResourceDataRaw(t, target.res.Schema, map[string]interface{}{})
				// d.State() returns nil on an empty id, which would make every
				// assertion below vacuously skip.
				d.SetId("pol-1")
				if err := d.Set(target.key, target.value); err != nil {
					t.Fatalf("%s: d.Set(%s): %v", target.where, target.key, err)
				}

				state := d.State()
				if state == nil {
					t.Fatalf("%s: no state was produced", target.where)
				}
				for field, want := range map[string]int64{"start": tc.start, "end": tc.end} {
					addr := target.prefix + "." + field
					stored, ok := state.Attributes[addr]
					if !ok {
						t.Fatalf("%s: %s is absent from the state", target.where, addr)
					}
					f, _, err := big.ParseFloat(stored, 10, 512, big.ToNearestEven)
					if err != nil {
						t.Fatalf("%s: %s = %q is not a number: %v", target.where, addr, stored, err)
					}
					got, acc := f.Int64()
					if acc != big.Exact || got != want {
						t.Fatalf("%s: %s stored as %q, which reads back as %d (%v), want exactly %d",
							target.where, addr, stored, got, acc, want)
					}
				}
			}
		})
	}
}

// --- the guard must actually RUN in the three Read paths -------------------
//
// The schema and helper tests above prove the guard is correct; these prove it
// is WIRED. Removing any of the three
// helpers.ValidateOpenIaaSReplicationPolicyLastRun calls from the Read paths
// turns the matching sub-test RED (the read then succeeds and writes a rounded
// timestamp into the state instead of failing closed).

// replicationPolicyBody renders one policy with the given lastRun timestamps.
func replicationPolicyBody(start, end int64) string {
	return fmt.Sprintf(
		`{"id":"pol-1","name":"nightly","lastRun":{"start":%d,"end":%d,"status":"SUCCESS"},"interval":{"hours":4,"minutes":30}}`,
		start, end)
}

// replicationReadPaths drives each Read through a stubbed API. A list route is
// answered with a one-element array, a single route with the bare object, so
// one handler serves all three.
func replicationReadPaths(t *testing.T, body string) map[string]func() (diag.Diagnostics, *schema.ResourceData) {
	t.Helper()

	newClient := func() *client.Client {
		return newAssignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if r.URL.Path == "/compute/v1/open_iaas/replication/configurations" {
				_, _ = w.Write([]byte("[" + body + "]"))
				return
			}
			_, _ = w.Write([]byte(body))
		})
	}

	return map[string]func() (diag.Diagnostics, *schema.ResourceData){
		"resource": func() (diag.Diagnostics, *schema.ResourceData) {
			d := resourceOpenIaasReplicationPolicy().TestResourceData()
			d.SetId("pol-1")
			return openIaasReplicationPolicyRead(context.Background(), d, newClient()), d
		},
		"data source": func() (diag.Diagnostics, *schema.ResourceData) {
			d := dataSourceOpenIaasReplicationPolicy().TestResourceData()
			_ = d.Set("id", "pol-1")
			return computeOpenIaaSReplicationPolicyRead(context.Background(), d, newClient()), d
		},
		"data sources": func() (diag.Diagnostics, *schema.ResourceData) {
			d := dataSourceOpenIaasReplicationPolicies().TestResourceData()
			return computeOpenIaaSReplicationPoliciesRead(context.Background(), d, newClient()), d
		},
	}
}

// TestReplicationReadsRefuseAnInexactTimestamp: an API answer the state cannot
// record exactly must fail the read, not be rounded into the state.
func TestReplicationReadsRefuseAnInexactTimestamp(t *testing.T) {
	body := replicationPolicyBody(helpers.MaxExactTimestampMs+1, 1759830012345)

	for where, run := range replicationReadPaths(t, body) {
		t.Run(where, func(t *testing.T) {
			diags, d := run()
			if !diags.HasError() {
				t.Fatalf("a timestamp beyond the exactly-storable range must fail the read, got %v", diags)
			}
			diagsContain(t, diags, "9007199254740993")

			state := d.State()

			if where == "resource" {
				// A read error is NOT a deletion: the resource must keep its id
				// so the next apply does not create a duplicate. Only the
				// refused answer must be absent.
				if d.Id() != "pol-1" {
					t.Fatalf("a refused read must not drop the resource from the state, id is now %q", d.Id())
				}
				if state == nil {
					t.Fatal("the resource lost its state entirely")
				}
				for addr, v := range state.Attributes {
					if strings.Contains(addr, "last_run") && v != "" && v != "0" {
						t.Fatalf("a refused read wrote %s = %q into the state", addr, v)
					}
				}
				return
			}

			// A data source holds nothing before its read, so a refused read
			// must leave it with no state at all. This pins the ORDERING: the
			// guard runs before d.SetId, not after it.
			if d.Id() != "" {
				t.Fatalf("a refused read set the data source id to %q: the guard must run before anything is written", d.Id())
			}
			if state != nil {
				t.Fatalf("a refused read produced state: %v", state.Attributes)
			}
		})
	}
}

// TestReplicationReadsRecordAContemporaryTimestamp is the anti-false-positive
// half: the guard must not refuse a normal answer, and the exact millisecond
// must reach the state through the whole decode -> validate -> flatten -> write
// chain.
func TestReplicationReadsRecordAContemporaryTimestamp(t *testing.T) {
	body := replicationPolicyBody(1759830000000, 1759830012345)

	for where, run := range replicationReadPaths(t, body) {
		t.Run(where, func(t *testing.T) {
			diags, d := run()
			if diags.HasError() {
				t.Fatalf("a normal read must succeed, got %v", diags)
			}
			prefix := "last_run.0"
			if where == "data sources" {
				prefix = "policies.0.last_run.0"
			}
			state := d.State()
			if state == nil {
				t.Fatal("no state was produced")
			}
			for field, want := range map[string]int64{"start": 1759830000000, "end": 1759830012345} {
				addr := prefix + "." + field
				stored, ok := state.Attributes[addr]
				if !ok {
					t.Fatalf("%s is absent from the state", addr)
				}
				f, _, err := big.ParseFloat(stored, 10, 512, big.ToNearestEven)
				if err != nil {
					t.Fatalf("%s = %q is not a number: %v", addr, stored, err)
				}
				if got, acc := f.Int64(); acc != big.Exact || got != want {
					t.Fatalf("%s stored as %q, reads back as %d (%v), want exactly %d", addr, stored, got, acc, want)
				}
			}
		})
	}
}

// TestReplicationResourceRefusalPreservesAPopulatedState closes the gap the
// refusal test above leaves: it starts from a state that ALREADY holds a valid
// last_run, so the assertion is "the previous values survived untouched", not
// merely "nothing was written into an empty state". A refusal that cleared,
// zeroed or half-wrote the block would be a state regression even though the
// read correctly returned an error.
func TestReplicationResourceRefusalPreservesAPopulatedState(t *testing.T) {
	const priorStart, priorEnd int64 = 1759740000000, 1759740009876

	d := resourceOpenIaasReplicationPolicy().TestResourceData()
	d.SetId("pol-1")
	if err := d.Set("name", "nightly"); err != nil {
		t.Fatalf("seeding name: %v", err)
	}
	if err := d.Set("last_run", []interface{}{map[string]interface{}{
		"start": priorStart, "end": priorEnd, "status": "SUCCESS",
	}}); err != nil {
		t.Fatalf("seeding last_run: %v", err)
	}

	c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(replicationPolicyBody(helpers.MaxExactTimestampMs+1, 1759830012345)))
	})

	diags := openIaasReplicationPolicyRead(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatalf("the read must be refused, got %v", diags)
	}

	if d.Id() != "pol-1" {
		t.Fatalf("a refused read dropped the resource from the state (id %q)", d.Id())
	}
	state := d.State()
	if state == nil {
		t.Fatal("a refused read wiped the state")
	}
	for field, want := range map[string]int64{"start": priorStart, "end": priorEnd} {
		addr := "last_run.0." + field
		stored, ok := state.Attributes[addr]
		if !ok {
			t.Fatalf("a refused read removed %s from the state", addr)
		}
		f, _, err := big.ParseFloat(stored, 10, 512, big.ToNearestEven)
		if err != nil {
			t.Fatalf("%s = %q is not a number: %v", addr, stored, err)
		}
		if got, acc := f.Int64(); acc != big.Exact || got != want {
			t.Fatalf("a refused read changed %s to %q (reads back as %d, %v), want the previous %d",
				addr, stored, got, acc, want)
		}
	}
	if status := state.Attributes["last_run.0.status"]; status != "SUCCESS" {
		t.Fatalf("a refused read changed last_run.0.status to %q", status)
	}
}

// TestReplicationPoliciesListingRefusesWhenOnlyOneItemIsOutOfRange pins the
// listing failure mode on the case that distinguishes "refuse the answer" from
// "skip the bad item": a valid policy FOLLOWED by an invalid one. Dropping the
// second would hand Terraform a listing that is shorter than the platform's
// with no evidence the policy is gone; keeping it would record a timestamp
// whose exactness is not guaranteed.
func TestReplicationPoliciesListingRefusesWhenOnlyOneItemIsOutOfRange(t *testing.T) {
	valid := `{"id":"pol-1","name":"nightly","lastRun":{"start":1759830000000,"end":1759830012345,"status":"SUCCESS"}}`
	invalid := `{"id":"pol-2","name":"hourly","lastRun":{"start":9007199254740993,"end":0,"status":"SUCCESS"}}`

	c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("[" + valid + "," + invalid + "]"))
	})

	d := dataSourceOpenIaasReplicationPolicies().TestResourceData()
	diags := computeOpenIaaSReplicationPoliciesRead(context.Background(), d, c)

	if !diags.HasError() {
		t.Fatalf("one out-of-range policy must refuse the whole listing, got %v", diags)
	}
	// The diagnostic must name the offending policy, not the first one read.
	diagsContain(t, diags, "pol-2")
	diagsContain(t, diags, "9007199254740993")

	if d.Id() != "" || d.State() != nil {
		t.Fatalf("a refused listing must leave nothing behind (id %q, state %v)", d.Id(), d.State())
	}
}
