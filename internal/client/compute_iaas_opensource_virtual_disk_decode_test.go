package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestOpenIaaSVirtualDiskDecodeContract pins the deserialization of the
// virtual-disk payload against the LIVE response shape (captured 2026-07-21
// on the DEV tenant while reproducing #488). The structs carry no JSON tags:
// decoding relies on Go's case-insensitive field matching, so a silent API
// field rename (readOnly -> read_only, …) would zero the field without any
// compile-time signal. IsPlatformManagedDisk keys the platform-drive
// exclusion on VirtualMachines[].ReadOnly — this test is the tripwire.
func TestOpenIaaSVirtualDiskDecodeContract(t *testing.T) {
	const payload = `{
		"id": "1f5c8276-99b2-4e16-9634-8f1ad51e4993",
		"internalId": "3fe6abe0-028a-463f-91d9-3e6781778608",
		"name": "XO CloudConfigDrive",
		"description": "",
		"size": 10485760,
		"usage": 12582912,
		"isSnapshot": false,
		"storageRepository": {
			"id": "2c882694-df86-4626-b456-4b89008bab4c",
			"name": "sr018"
		},
		"virtualMachines": [
			{
				"id": "e3428d87-4594-41ab-bbd8-5fdae48e1faa",
				"name": "some-vm",
				"readOnly": true,
				"connected": false
			}
		]
	}`

	resp := &http.Response{Body: io.NopCloser(strings.NewReader(payload))}
	var disk OpenIaaSVirtualDisk
	if err := decodeBody(resp, &disk); err != nil {
		t.Fatalf("decodeBody: %v", err)
	}

	if disk.ID != "1f5c8276-99b2-4e16-9634-8f1ad51e4993" {
		t.Errorf("ID not decoded: %q", disk.ID)
	}
	if disk.Name != "XO CloudConfigDrive" {
		t.Errorf("Name not decoded: %q", disk.Name)
	}
	if disk.Size != 10485760 {
		t.Errorf("Size not decoded: %d", disk.Size)
	}
	if disk.StorageRepository.ID != "2c882694-df86-4626-b456-4b89008bab4c" {
		t.Errorf("StorageRepository.ID not decoded: %q", disk.StorageRepository.ID)
	}
	if len(disk.VirtualMachines) != 1 {
		t.Fatalf("VirtualMachines not decoded: %#v", disk.VirtualMachines)
	}
	vbd := disk.VirtualMachines[0]
	if vbd.ID != "e3428d87-4594-41ab-bbd8-5fdae48e1faa" {
		t.Errorf("VBD ID not decoded: %q", vbd.ID)
	}
	// The load-bearing field: a decode regression here silently disables the
	// read-only arm of the platform-drive exclusion (#255/#488).
	if !vbd.ReadOnly {
		t.Error("VirtualMachines[0].ReadOnly was not decoded from \"readOnly\": the platform-drive discriminator would silently stop matching")
	}
	if vbd.Connected {
		t.Error("VirtualMachines[0].Connected mis-decoded: expected false")
	}
}
