package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestFlattenVirtualMachineExtraConfigIsAStringMap is the dedicated #241-area
// content test. The VM `extra_config` attribute is a TypeMap of string values.
// The historical failure mode emits it as a LIST of {key,value} blocks instead
// of a map, and d.Set then fails with "source data must be an array or slice,
// got map" / the reverse, breaking the whole VM datasource read (#241).
//
// The structural walker checks the fully-filled output fits the schema; this
// test pins the CONTENT: extra_config must be a map[string]string keyed by the
// VMware config key, mapping to its value, with every entry preserved.
func TestFlattenVirtualMachineExtraConfigIsAStringMap(t *testing.T) {
	vm := &client.VirtualMachine{
		Name: "vm-1",
		ExtraConfig: []client.VirtualMachineExtraConfig{
			{Key: "disk.enableUUID", Value: "TRUE"},
			{Key: "guestinfo.foo", Value: "bar"},
			{Key: "pciPassthru.64bitMMioSizeGB", Value: "128"},
		},
	}

	got := FlattenVirtualMachine(vm)

	ec, ok := got["extra_config"].(map[string]string)
	if !ok {
		t.Fatalf("extra_config has type %T, want map[string]string; a list shape here is the #241 break (d.Set 'source data must be an array or slice, got map')", got["extra_config"])
	}

	want := map[string]string{
		"disk.enableUUID":             "TRUE",
		"guestinfo.foo":               "bar",
		"pciPassthru.64bitMMioSizeGB": "128",
	}
	if len(ec) != len(want) {
		t.Fatalf("extra_config has %d entries, want %d: got %#v", len(ec), len(want), ec)
	}
	for k, v := range want {
		if ec[k] != v {
			t.Errorf("extra_config[%q] = %q, want %q", k, ec[k], v)
		}
	}
}

// TestFlattenVirtualMachineExtraConfigEmptyIsEmptyMap proves an empty
// ExtraConfig flattens to an empty, non-nil map (never a nil and never a list),
// so the attribute stays a stable map shape on a VM with no extra config.
func TestFlattenVirtualMachineExtraConfigEmptyIsEmptyMap(t *testing.T) {
	got := FlattenVirtualMachine(&client.VirtualMachine{Name: "vm-empty"})

	ec, ok := got["extra_config"].(map[string]string)
	if !ok {
		t.Fatalf("extra_config has type %T, want map[string]string even when empty", got["extra_config"])
	}
	if ec == nil {
		t.Errorf("extra_config is a nil map; expected a non-nil empty map[string]string")
	}
	if len(ec) != 0 {
		t.Errorf("extra_config = %#v, want empty", ec)
	}
}

// TestFlattenVirtualMachineExtraConfigLastWriteWins documents the duplicate-key
// behavior: ExtraConfig is a list and the map collapses duplicates, with the
// last occurrence winning. This pins the intended semantics so a future change
// (e.g. first-wins, or emitting a list) is a deliberate, reviewed decision.
func TestFlattenVirtualMachineExtraConfigLastWriteWins(t *testing.T) {
	vm := &client.VirtualMachine{
		ExtraConfig: []client.VirtualMachineExtraConfig{
			{Key: "dup", Value: "first"},
			{Key: "dup", Value: "last"},
		},
	}
	ec := FlattenVirtualMachine(vm)["extra_config"].(map[string]string)
	if ec["dup"] != "last" {
		t.Errorf("extra_config[dup] = %q, want last (last-write-wins)", ec["dup"])
	}
}

// TestFlattenVirtualMachineNestedBlocksAreConditional pins the conditional
// emission of the optional single-element blocks (storage, boot_options,
// replication_config). They are emitted as a one-element list ONLY when the
// source carries real data, and as an empty (non-nil) list otherwise. A
// regression that always emits a bare element, or that emits a nil, would
// surface as VM drift; this test catches both directions.
func TestFlattenVirtualMachineNestedBlocksAreConditional(t *testing.T) {
	// (1) bare VM: every optional block is an empty, non-nil list.
	bare := FlattenVirtualMachine(&client.VirtualMachine{})
	for _, key := range []string{"storage", "boot_options", "replication_config", "triggered_alarms"} {
		list, ok := bare[key].([]map[string]interface{})
		if !ok {
			t.Fatalf("%s has type %T, want []map[string]interface{}", key, bare[key])
		}
		if list == nil {
			t.Errorf("%s is a nil slice on a bare VM; expected a non-nil empty list", key)
		}
		if len(list) != 0 {
			t.Errorf("%s = %v on a bare VM, want empty", key, list)
		}
	}

	// (2) VM with real storage and boot options: the blocks materialize with
	// the right content.
	vm := &client.VirtualMachine{}
	vm.Storage.Committed = 10
	vm.Storage.Uncommitted = 5
	vm.BootOptions.Firmware = "efi"
	vm.BootOptions.BootDelay = 200

	got := FlattenVirtualMachine(vm)
	storage := got["storage"].([]map[string]interface{})
	if len(storage) != 1 {
		t.Fatalf("storage has %d elements, want 1", len(storage))
	}
	assertEq(t, "storage.committed", storage[0]["committed"], 10)
	assertEq(t, "storage.uncommitted", storage[0]["uncommitted"], 5)

	boot := got["boot_options"].([]map[string]interface{})
	if len(boot) != 1 {
		t.Fatalf("boot_options has %d elements, want 1", len(boot))
	}
	assertEq(t, "boot_options.firmware", boot[0]["firmware"], "efi")
	assertEq(t, "boot_options.boot_delay", boot[0]["boot_delay"], 200)
}

// TestFlattenVirtualMachineTriggeredAlarmsOrder pins that triggered_alarms
// preserves source order and content, since list order is observable in the
// state and an unstable order is permanent drift.
func TestFlattenVirtualMachineTriggeredAlarmsOrder(t *testing.T) {
	vm := &client.VirtualMachine{
		TriggeredAlarms: []client.VirtualMachineTriggeredAlarm{
			{ID: "a1", Status: "red"},
			{ID: "a2", Status: "yellow"},
		},
	}
	alarms := FlattenVirtualMachine(vm)["triggered_alarms"].([]map[string]interface{})
	if len(alarms) != 2 {
		t.Fatalf("triggered_alarms has %d elements, want 2", len(alarms))
	}
	assertEq(t, "triggered_alarms[0].id", alarms[0]["id"], "a1")
	assertEq(t, "triggered_alarms[0].status", alarms[0]["status"], "red")
	assertEq(t, "triggered_alarms[1].id", alarms[1]["id"], "a2")
}
