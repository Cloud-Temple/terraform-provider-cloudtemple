package helpers

import (
	"encoding/json"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// addressBlock extracts the single {ipv4, ipv6} map the VM flatten emits under
// the "addresses" key, failing loudly on any shape drift so the assertion can
// never pass vacuously.
func addressBlock(t *testing.T, flat map[string]interface{}) map[string]interface{} {
	t.Helper()
	raw, ok := flat["addresses"]
	if !ok {
		t.Fatalf("flatten output has no %q key", "addresses")
	}
	list, ok := raw.([]map[string]interface{})
	if !ok || len(list) != 1 {
		t.Fatalf("addresses block = %#v, want a 1-element []map[string]interface{}", raw)
	}
	return list[0]
}

// TestFlattenOpenIaaSVirtualMachineAddressesCompositeKeys is the real proof for
// the Volet A fix (#238): it decodes a REAL API-shaped payload whose addresses
// object is keyed by the composite "<device>/<family>/<index>" form
// ("0/ipv4/0", "0/ipv6/0") into the client struct, then flattens it and asserts
// the {ipv4, ipv6} state block is POPULATED with those exact values.
//
// RED EVIDENCE (documented, not committed): against the OLD client decode —
//
//	Addresses struct { IPv6 string; IPv4 string }
//
// the composite JSON keys bind to NO field, so vm.Addresses is the zero struct
// and the flatten emits ipv4="" / ipv6="". Both assertions below then fail:
//
//	ipv4 = "", want "10.0.0.5"
//	ipv6 = "", want "fe80::1"
//
// A test that hand-built the new map[string]string would NOT exercise the
// decode and would be complacent; this one starts from the wire bytes.
func TestFlattenOpenIaaSVirtualMachineAddressesCompositeKeys(t *testing.T) {
	const payload = `{
		"name": "vm-prod",
		"addresses": {
			"0/ipv4/0": "10.0.0.5",
			"0/ipv6/0": "fe80::1"
		}
	}`

	var vm client.OpenIaaSVirtualMachine
	if err := json.Unmarshal([]byte(payload), &vm); err != nil {
		t.Fatalf("decoding the API payload failed: %s", err)
	}

	got := addressBlock(t, FlattenOpenIaaSVirtualMachine(&vm))
	assertEq(t, "ipv4", got["ipv4"], "10.0.0.5")
	assertEq(t, "ipv6", got["ipv6"], "fe80::1")
}

// TestFlattenOpenIaaSVirtualMachineAddressesEmpty pins the nil/empty-map case:
// a VM with no addresses must flatten to ipv4="" / ipv6="" WITHOUT panicking.
func TestFlattenOpenIaaSVirtualMachineAddressesEmpty(t *testing.T) {
	// nil map (zero-valued struct path).
	got := addressBlock(t, FlattenOpenIaaSVirtualMachine(&client.OpenIaaSVirtualMachine{}))
	assertEq(t, "ipv4", got["ipv4"], "")
	assertEq(t, "ipv6", got["ipv6"], "")

	// explicitly empty map.
	got = addressBlock(t, FlattenOpenIaaSVirtualMachine(&client.OpenIaaSVirtualMachine{
		Addresses: map[string]string{},
	}))
	assertEq(t, "ipv4", got["ipv4"], "")
	assertEq(t, "ipv6", got["ipv6"], "")
}

// TestFlattenOpenIaaSVirtualMachineAddressesFallback pins the fallback path:
// when the exact "0/<family>/0" key is absent, the FIRST "*/<family>/*" key in
// LEXICAL order is used (deterministic, never map-iteration-order dependent),
// and a family with no matching key collapses to "".
func TestFlattenOpenIaaSVirtualMachineAddressesFallback(t *testing.T) {
	// No "0/ipv4/0", but two ipv4 keys on other devices/indices. Sorted
	// lexically, "1/ipv4/0" < "2/ipv4/0", so the .9 address must win.
	// ipv6 has no key at all and must be "".
	got := addressBlock(t, FlattenOpenIaaSVirtualMachine(&client.OpenIaaSVirtualMachine{
		Addresses: map[string]string{
			"2/ipv4/0": "10.0.0.20",
			"1/ipv4/0": "10.0.0.9",
		},
	}))
	// Mutation proof: relying on Go's randomized map order (no sort) makes this
	// flake; dropping the fallback entirely makes it RED with "".
	assertEq(t, "ipv4", got["ipv4"], "10.0.0.9")
	assertEq(t, "ipv6", got["ipv6"], "")
}

func TestIsPlatformManagedDisk(t *testing.T) {
	const vmID = "vm-1"

	tests := []struct {
		name                 string
		disk                 *client.OpenIaaSVirtualDisk
		cloudInitProvisioned bool
		want                 bool
	}{
		{
			name: "read-only XO config drive on this VM is excluded",
			disk: &client.OpenIaaSVirtualDisk{
				Name: CloudConfigDriveName,
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: true},
				},
			},
			want: true,
		},
		{
			// #488: the platform attaches the config drive with a READ-WRITE
			// VBD on the deploy path (live evidence, 2026-07-21) — the
			// resource's own cloud-init provisioning is the second evidence.
			name: "writable XO config drive on this VM is excluded when cloud-init is provisioned (#488)",
			disk: &client.OpenIaaSVirtualDisk{
				Name: CloudConfigDriveName,
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: false},
				},
			},
			cloudInitProvisioned: true,
			want:                 true,
		},
		{
			name: "writable user disk with the colliding name stays managed without cloud-init",
			disk: &client.OpenIaaSVirtualDisk{
				Name: CloudConfigDriveName,
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: false},
				},
			},
			want: false,
		},
		{
			name: "read-only disk with another name stays managed",
			disk: &client.OpenIaaSVirtualDisk{
				Name: "data-disk",
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: true},
				},
			},
			want: false,
		},
		{
			name: "disk with another name stays managed even with cloud-init provisioned",
			disk: &client.OpenIaaSVirtualDisk{
				Name: "data-disk",
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: false},
				},
			},
			cloudInitProvisioned: true,
			want:                 false,
		},
		{
			name: "XO-named disk read-only on another VM only stays managed",
			disk: &client.OpenIaaSVirtualDisk{
				Name: CloudConfigDriveName,
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: "vm-other", ReadOnly: true},
				},
			},
			want: false,
		},
		{
			// #488 F2 guard: the cloud-init arm must NOT bypass the
			// VBD-on-this-VM requirement (fail-safe keep).
			name: "XO-named disk without a VBD on this VM stays managed even with cloud-init provisioned",
			disk: &client.OpenIaaSVirtualDisk{
				Name: CloudConfigDriveName,
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: "vm-other", ReadOnly: false},
				},
			},
			cloudInitProvisioned: true,
			want:                 false,
		},
		{
			name: "XO-named disk without any VBD stays managed (fail-safe)",
			disk: &client.OpenIaaSVirtualDisk{
				Name: CloudConfigDriveName,
			},
			cloudInitProvisioned: true,
			want:                 false,
		},
		{
			name:                 "a nil disk is not platform-managed and must not panic (#320)",
			disk:                 nil,
			cloudInitProvisioned: true,
			want:                 false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPlatformManagedDisk(tt.disk, vmID, tt.cloudInitProvisioned); got != tt.want {
				t.Errorf("IsPlatformManagedDisk() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsPlatformManagedDiskEmptyVMID pins the degenerate empty-id guard: the
// zero-value VBD (ID "") must never accidentally match an empty resource id.
func TestIsPlatformManagedDiskEmptyVMID(t *testing.T) {
	disk := &client.OpenIaaSVirtualDisk{Name: CloudConfigDriveName}
	if IsPlatformManagedDisk(disk, "", true) {
		t.Fatal("an empty virtual machine id must never match the zero-value VBD")
	}
}

// TestFlattenOpenIaaSOSDisksDataSkipsConfigDrive pins the #488 create-time
// capture fix in both listing orders: with cloud-init provisioned, the
// platform config drive (RW VBD) is skipped and the system disk survives as
// the FIRST kept entry, whatever position the API listed the drive at.
func TestFlattenOpenIaaSOSDisksDataSkipsConfigDrive(t *testing.T) {
	const vmID = "vm-1"
	system := &client.OpenIaaSVirtualDisk{
		ID:   "disk-system",
		Name: "ubuntu-24.04-disk-0",
		VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
			{ID: vmID, ReadOnly: false},
		},
	}
	drive := &client.OpenIaaSVirtualDisk{
		ID:   "disk-drive",
		Name: CloudConfigDriveName,
		VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
			{ID: vmID, ReadOnly: false},
		},
	}

	for name, order := range map[string][]*client.OpenIaaSVirtualDisk{
		"system disk first":  {system, drive},
		"config drive first": {drive, system},
	} {
		t.Run(name, func(t *testing.T) {
			got := FlattenOpenIaaSOSDisksData(order, vmID, true)
			if len(got) != 1 {
				t.Fatalf("expected exactly the system disk to survive, got %d entries: %#v", len(got), got)
			}
			if id := got[0].(map[string]interface{})["id"]; id != "disk-system" {
				t.Fatalf("expected the system disk at index 0, got id %v", id)
			}
		})
	}

	// Without cloud-init the RW drive is NOT skipped (pre-#488 behavior kept
	// for non-cloud-init VMs: name alone is never enough).
	got := FlattenOpenIaaSOSDisksData([]*client.OpenIaaSVirtualDisk{system, drive}, vmID, false)
	if len(got) != 2 {
		t.Fatalf("without cloud-init both disks must stay managed, got %d entries", len(got))
	}
}

// TestFlattenOpenIaaSOSDiskDataNilIsSafe pins the #320 fix (ported to rc as
// defense-in-depth on top of classifyOSDiskOnRead): a nil OS disk (the API maps
// a deleted/forbidden disk to nil) must NOT be dereferenced. Without the guard
// this panics on osDisk.VirtualMachines, crashing `terraform plan`.
func TestFlattenOpenIaaSOSDiskDataNilIsSafe(t *testing.T) {
	if got := FlattenOpenIaaSOSDiskData(nil, "vm-1"); got != nil {
		t.Fatalf("expected nil for a nil os disk, got %#v", got)
	}
}

// TestFlattenOpenIaaSOSDisksDataSkipsNil pins that the plural flatten skips nil
// disk entries (rather than dereferencing or appending a nil map) and keeps the
// valid ones.
func TestFlattenOpenIaaSOSDisksDataSkipsNil(t *testing.T) {
	valid := &client.OpenIaaSVirtualDisk{
		ID:                "disk-1",
		Name:              "data",
		StorageRepository: client.BaseObject{ID: "sr-1"},
		VirtualMachines:   []client.OpenIaaSVirtualDiskConnection{{ID: "vm-1", Connected: true}},
	}

	got := FlattenOpenIaaSOSDisksData([]*client.OpenIaaSVirtualDisk{nil, valid, nil}, "vm-1", false)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 disk (nils skipped), got %d: %#v", len(got), got)
	}
	m, ok := got[0].(map[string]interface{})
	if !ok || m["id"] != "disk-1" || m["connected"] != true {
		t.Fatalf("unexpected flattened disk: %#v", got[0])
	}
}
