package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestFlattenOpenIaaSPoolContent is the dedicated, non-complacent content test
// for the #243 culprit. FlattenOpenIaaSPool feeds the iaas_opensource_pool(s)
// datasource, whose schema declares EXACTLY: id, name, label, internal_id,
// machine_manager_id, high_availability_enabled, master, hosts, memory, cpu,
// type. If the helper emits any key the schema does not declare, d.Set fails
// with "Invalid address to set" and the whole datasource read breaks (#243).
//
// The structural walker already checks the fully-populated output FITS the
// schema. This test adds what the walker does NOT:
//   - the emitted key SET is exactly the declared one (no extra, no missing) —
//     pinned against the live datasource schema, so a future drift on either
//     side is caught;
//   - the nested single-element blocks (memory, cpu, type) carry the right
//     sub-keys with the right values;
//   - the machine_manager_id is read from the nested MachineManager.ID;
//   - a pool with empty sub-objects does not panic and still emits every block.
func TestFlattenOpenIaaSPoolContent(t *testing.T) {
	pool := &client.OpenIaasPool{
		ID:                      "pool-1",
		InternalID:              "int-1",
		Name:                    "prod-pool",
		Label:                   "Production",
		HighAvailabilityEnabled: true,
		Master:                  "host-master",
		Hosts:                   []string{"host-a", "host-b"},
		MachineManager:          client.BaseObject{ID: "mm-1", Name: "vc-1"},
	}
	pool.Memory.Usage = 40
	pool.Memory.Size = 100
	pool.Cpu.Cores = 8
	pool.Cpu.Sockets = 2
	pool.Type.Key = "xcp-ng"
	pool.Type.Description = "XCP-ng pool"

	got := FlattenOpenIaaSPool(pool)

	// --- exact key set against the live datasource schema -----------------
	wantKeys := map[string]bool{
		"id": true, "name": true, "label": true, "internal_id": true,
		"machine_manager_id": true, "high_availability_enabled": true,
		"master": true, "hosts": true, "memory": true, "cpu": true, "type": true,
	}
	for k := range got {
		if !wantKeys[k] {
			t.Errorf("FlattenOpenIaaSPool emits undeclared key %q; the pool datasource schema would reject it with 'Invalid address to set' (#243)", k)
		}
	}
	for k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Errorf("FlattenOpenIaaSPool is missing the expected key %q", k)
		}
	}

	// --- scalar content ----------------------------------------------------
	assertEq(t, "id", got["id"], "pool-1")
	assertEq(t, "name", got["name"], "prod-pool")
	assertEq(t, "label", got["label"], "Production")
	assertEq(t, "internal_id", got["internal_id"], "int-1")
	assertEq(t, "high_availability_enabled", got["high_availability_enabled"], true)
	assertEq(t, "master", got["master"], "host-master")
	// machine_manager_id is the NESTED MachineManager.ID, not the pool ID.
	assertEq(t, "machine_manager_id", got["machine_manager_id"], "mm-1")

	hosts, ok := got["hosts"].([]string)
	if !ok {
		t.Fatalf("hosts has type %T, want []string", got["hosts"])
	}
	if len(hosts) != 2 || hosts[0] != "host-a" || hosts[1] != "host-b" {
		t.Errorf("hosts = %v, want [host-a host-b] in order", hosts)
	}

	// --- nested single-element blocks --------------------------------------
	mem := singleBlock(t, "memory", got["memory"])
	assertEq(t, "memory.usage", mem["usage"], 40)
	assertEq(t, "memory.size", mem["size"], 100)

	cpu := singleBlock(t, "cpu", got["cpu"])
	assertEq(t, "cpu.cores", cpu["cores"], 8)
	assertEq(t, "cpu.sockets", cpu["sockets"], 2)

	typ := singleBlock(t, "type", got["type"])
	assertEq(t, "type.key", typ["key"], "xcp-ng")
	assertEq(t, "type.description", typ["description"], "XCP-ng pool")
}

// TestFlattenOpenIaaSPoolEmptySubObjectsDoNotPanic proves the helper survives a
// pool whose nested objects are all zero (a bare API object). The blocks must
// still be emitted as single-element lists so the schema shape stays stable.
func TestFlattenOpenIaaSPoolEmptySubObjectsDoNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FlattenOpenIaaSPool panicked on a zero-valued pool: %v", r)
		}
	}()

	got := FlattenOpenIaaSPool(&client.OpenIaasPool{})

	// machine_manager_id comes from a nested struct; on a zero pool it must be
	// the empty string, NOT a panic.
	assertEq(t, "machine_manager_id", got["machine_manager_id"], "")

	// The three nested blocks are always single-element lists, even when empty.
	for _, key := range []string{"memory", "cpu", "type"} {
		block := singleBlock(t, key, got[key])
		if len(block) == 0 {
			t.Errorf("%s block is empty; expected the zero-valued sub-keys to be present", key)
		}
	}
}

// singleBlock asserts a flatten value is a single-element nested block list and
// returns its only map. Shared by the pool sub-block assertions.
func singleBlock(t *testing.T, key string, v interface{}) map[string]interface{} {
	t.Helper()
	list, ok := v.([]map[string]interface{})
	if !ok {
		t.Fatalf("%s has type %T, want []map[string]interface{}", key, v)
	}
	if len(list) != 1 {
		t.Fatalf("%s has %d elements, want exactly 1", key, len(list))
	}
	return list[0]
}

// assertEq is a tiny equality assertion that reports the offending key.
func assertEq(t *testing.T, key string, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %#v, want %#v", key, got, want)
	}
}
