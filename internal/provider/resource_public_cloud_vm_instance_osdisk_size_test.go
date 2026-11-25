package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// os_disk carries the ONE user-settable attribute touched by the capacity rename
// (issue #524), so it is the only one kept under a deprecation window instead of
// renamed outright: getting it wrong could resize a client's system disk.
//
// These tests drive the real CustomizeDiff through Resource.Diff, which is what
// proves the RAW-config path is genuinely exercised — the rules cannot read the
// diff/state view, both size attributes being Optional+Computed.

// vmInstanceConfig builds a ResourceConfig carrying a cty value, so that
// GetRawConfig() inside CustomizeDiff sees the declared config rather than a
// null. NewResourceConfigShimmed is what populates the cty side.
func vmInstanceConfigValue(t *testing.T, osDisk cty.Value) cty.Value {
	t.Helper()
	r := resourcePublicCloudVMInstance()
	block := r.CoreConfigSchema()
	obj := map[string]cty.Value{
		"name":                 cty.StringVal("web"),
		"availability_zone_id": cty.StringVal("11111111-1111-1111-1111-111111111111"),
		"image_id":             cty.StringVal("22222222-2222-2222-2222-222222222222"),
		"instance_family_id":   cty.StringVal("33333333-3333-3333-3333-333333333333"),
		"cpu":                  cty.NumberIntVal(2),
		"memory":               cty.NumberIntVal(4),
		"power_state":          cty.StringVal("off"),
		"os_network_adapter": cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
			"device_index": cty.NumberIntVal(0),
			"network_id":   cty.StringVal("55555555-5555-5555-5555-555555555555"),
			"id":           cty.NullVal(cty.String),
			"ip_address":   cty.NullVal(cty.String),
			"mac_address":  cty.NullVal(cty.String),
		})}),
	}
	// Fill every remaining attribute of the block with a null of its own type, so
	// the object matches the schema exactly.
	full := map[string]cty.Value{}
	for name, attr := range block.Attributes {
		if v, ok := obj[name]; ok {
			full[name] = v
			continue
		}
		full[name] = cty.NullVal(attr.Type)
	}
	for name, nested := range block.BlockTypes {
		if v, ok := obj[name]; ok {
			full[name] = v
			continue
		}
		if name == "os_disk" {
			full[name] = osDisk
			continue
		}
		full[name] = cty.NullVal(nested.ImpliedType())
	}
	if _, ok := full["os_disk"]; !ok {
		full["os_disk"] = osDisk
	}
	return cty.ObjectVal(full)
}

// vmInstancePlan runs the resource's real Diff (CustomizeDiff included) the way
// the plugin server does: the declared config is carried BOTH as the shimmed
// ResourceConfig and as the prior state's RawConfig, because schemaMap.Diff
// propagates the raw config from the state (`result.RawConfig = s.RawConfig`).
// Passing it only as a ResourceConfig would leave GetRawConfig() null and
// silently skip every rule under test.
func vmInstancePlan(t *testing.T, state *terraform.InstanceState, osDisk cty.Value) (*terraform.InstanceDiff, error) {
	t.Helper()
	r := resourcePublicCloudVMInstance()
	configVal := vmInstanceConfigValue(t, osDisk)
	if state == nil {
		state = &terraform.InstanceState{Attributes: map[string]string{}}
	}
	state.RawConfig = configVal
	return r.Diff(context.Background(), state, terraform.NewResourceConfigShimmed(configVal, r.CoreConfigSchema()), nil)
}

// osDiskBlock builds an os_disk list value declaring the given sizes; pass a
// null to leave a spelling undeclared.
func osDiskBlock(t *testing.T, sizeGib, sizeGb cty.Value) cty.Value {
	t.Helper()
	r := resourcePublicCloudVMInstance()
	elemType := r.CoreConfigSchema().BlockTypes["os_disk"].Block.ImpliedType()
	attrs := map[string]cty.Value{}
	for name, ty := range elemType.AttributeTypes() {
		attrs[name] = cty.NullVal(ty)
	}
	attrs["size_gib"] = sizeGib
	attrs["size_gb"] = sizeGb
	return cty.ListVal([]cty.Value{cty.ObjectVal(attrs)})
}

// existingVMState is the state of a VM already created, with a 38 GiB system
// disk recorded under BOTH spellings — which is what the read path writes.
func existingVMState(sizeGib, sizeGb string) *terraform.InstanceState {
	return &terraform.InstanceState{
		ID: "vm-1",
		Attributes: map[string]string{
			"id":                   "vm-1",
			"name":                 "web",
			"availability_zone_id": "11111111-1111-1111-1111-111111111111",
			"image_id":             "22222222-2222-2222-2222-222222222222",
			"instance_family_id":   "33333333-3333-3333-3333-333333333333",
			"cpu":                  "2",
			"memory":               "4",
			"power_state":          "off",
			"os_disk.#":            "1",
			"os_disk.0.id":         "sys",
			"os_disk.0.size_gib":   sizeGib,
			"os_disk.0.size_gb":    sizeGb,
			"os_disk.0.is_primary": "true",
			// os_network_adapter is ForceNew: leaving it out of the state would
			// make every plan a replacement, and schemaMap.Diff then REPLAYS
			// CustomizeDiff with no state at all ("Reset the data to not contain
			// state"), which silently turns an update case into a create case.
			"os_network_adapter.#":              "1",
			"os_network_adapter.0.device_index": "0",
			"os_network_adapter.0.network_id":   "55555555-5555-5555-5555-555555555555",
		},
	}
}

// TestOSDiskBothSpellingsRejected pins the mutual exclusion. The two attributes
// are the same value under two names, so declaring both can only express (or
// hide) a contradiction. Rejected at PLAN time, before any write.
//
// Mutation: remove the declared.both() guard and this goes RED.
func TestOSDiskBothSpellingsRejected(t *testing.T) {
	_, err := vmInstancePlan(t, existingVMState("38", "38"), osDiskBlock(t, cty.NumberIntVal(45), cty.NumberIntVal(45)))
	if err == nil {
		t.Fatal("declaring both size_gib and size_gb must be rejected at plan time")
	}
	if !strings.Contains(err.Error(), "size_gib") || !strings.Contains(err.Error(), "size_gb") {
		t.Fatalf("the error must name both attributes to be actionable, got: %v", err)
	}
}

// TestOSDiskGrowRuleAppliesToEitherSpelling proves the deprecation window is real
// on BOTH sides: the grow-only / VM-must-be-stopped rules must fire whichever
// name the user drives, otherwise the deprecated spelling would become an
// unguarded path to a resize.
//
// This is the case that would silently regress if the update path only watched
// the current spelling.
func TestOSDiskGrowRuleAppliesToEitherSpelling(t *testing.T) {
	cases := []struct {
		name    string
		osDisk  cty.Value
		wantErr string
	}{
		{
			name:    "shrink through the current spelling is rejected",
			osDisk:  osDiskBlock(t, cty.NumberIntVal(20), cty.NullVal(cty.Number)),
			wantErr: "grow-only",
		},
		{
			name:    "shrink through the DEPRECATED spelling is rejected too",
			osDisk:  osDiskBlock(t, cty.NullVal(cty.Number), cty.NumberIntVal(20)),
			wantErr: "grow-only",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := vmInstancePlan(t, existingVMState("38", "38"), tc.osDisk)
			if err == nil {
				t.Fatalf("expected the change to be rejected")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want an error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// TestOSDiskGrowAcceptedThroughEitherSpelling is the GREEN counterpart: a
// legitimate grow on a stopped VM must plan cleanly under either name. Without
// it, the test above would be satisfied by a rule that rejects everything.
func TestOSDiskGrowAcceptedThroughEitherSpelling(t *testing.T) {
	cases := []struct {
		name   string
		osDisk cty.Value
	}{
		{"current spelling", osDiskBlock(t, cty.NumberIntVal(60), cty.NullVal(cty.Number))},
		{"deprecated spelling", osDiskBlock(t, cty.NullVal(cty.Number), cty.NumberIntVal(60))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diff, err := vmInstancePlan(t, existingVMState("38", "38"), tc.osDisk)
			if err != nil {
				t.Fatalf("a grow on a stopped VM must plan cleanly, got: %v", err)
			}
			if diff == nil {
				t.Fatal("expected a diff for the grow")
			}
		})
	}
}

// TestOSDiskSizeRejectedAtCreateUnderEitherSpelling pins that the "not settable
// at create" rule was not weakened by the second attribute: the VM is created
// with the image's disk size, and create never extends it.
func TestOSDiskSizeRejectedAtCreateUnderEitherSpelling(t *testing.T) {
	cases := []struct {
		name   string
		osDisk cty.Value
	}{
		{"current spelling", osDiskBlock(t, cty.NumberIntVal(80), cty.NullVal(cty.Number))},
		{"deprecated spelling", osDiskBlock(t, cty.NullVal(cty.Number), cty.NumberIntVal(80))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// An empty state == create.
			_, err := vmInstancePlan(t, nil, tc.osDisk)
			if err == nil {
				t.Fatal("setting an os_disk size at create must be rejected")
			}
			if !strings.Contains(err.Error(), "cannot be set when creating") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestOSDiskCreateWithoutSizeIsAccepted is the counterpart proving the create
// rejection is targeted, not a blanket refusal of the block.
func TestOSDiskCreateWithoutSizeIsAccepted(t *testing.T) {
	r := resourcePublicCloudVMInstance()
	noBlock := cty.NullVal(r.CoreConfigSchema().BlockTypes["os_disk"].ImpliedType())
	if _, err := vmInstancePlan(t, nil, noBlock); err != nil {
		t.Fatalf("creating without an os_disk size must be accepted, got: %v", err)
	}
}
