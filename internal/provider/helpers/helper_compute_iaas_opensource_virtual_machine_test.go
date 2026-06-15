package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestFlattenOpenIaaSOSDiskDataNilIsSafe pins the #320 fix: a nil OS disk (the
// API maps a deleted/forbidden disk to a nil read) must NOT be dereferenced.
// Without the guard this panics (SIGSEGV on osDisk.VirtualMachines), crashing
// `terraform plan` while refreshing a VM whose OS disk disappeared out-of-band.
func TestFlattenOpenIaaSOSDiskDataNilIsSafe(t *testing.T) {
	if got := FlattenOpenIaaSOSDiskData(nil, "vm-1"); got != nil {
		t.Fatalf("expected nil for a nil os disk, got %#v", got)
	}
}

// TestFlattenOpenIaaSOSDisksDataSkipsNil pins that the plural flatten skips nil
// disk entries (rather than appending a nil map or panicking) and keeps the
// valid ones.
func TestFlattenOpenIaaSOSDisksDataSkipsNil(t *testing.T) {
	valid := &client.OpenIaaSVirtualDisk{
		ID:                "disk-1",
		Name:              "data",
		StorageRepository: client.BaseObject{ID: "sr-1"},
		VirtualMachines:   []client.OpenIaaSVirtualDiskConnection{{ID: "vm-1", Connected: true}},
	}

	got := FlattenOpenIaaSOSDisksData([]*client.OpenIaaSVirtualDisk{nil, valid, nil}, "vm-1")
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 disk (nils skipped), got %d: %#v", len(got), got)
	}
	m, ok := got[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a map entry, got %#v", got[0])
	}
	if m["id"] != "disk-1" || m["connected"] != true {
		t.Fatalf("unexpected flattened disk: %#v", m)
	}
}
