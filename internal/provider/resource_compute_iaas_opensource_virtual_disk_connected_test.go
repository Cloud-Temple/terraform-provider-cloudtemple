package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// diskWithVBD builds a virtual disk whose VBD list attaches it to vmID with the
// given runtime plug state (Connected).
func diskWithVBD(diskID, vmID string, connected bool) *client.OpenIaaSVirtualDisk {
	return &client.OpenIaaSVirtualDisk{
		ID: diskID,
		VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
			{ID: vmID, Connected: connected},
		},
	}
}

// TestResolveOpenIaaSDiskConnected pins the power-off drift fix (#350):
// `connected` mirrors the runtime VBD plug state ONLY while the VM runs; on a
// halted/paused VM (where every VBD is reported unplugged) it preserves the
// known intent; a VBD no longer attached to the configured VM and an unknown
// power state both fail closed.
func TestResolveOpenIaaSDiskConnected(t *testing.T) {
	const diskID, vmID = "disk-1", "vm-1"

	t.Run("VM running, runtime unplugged, intent connected -> mirror runtime false (real disconnect surfaces)", func(t *testing.T) {
		got, err := resolveOpenIaaSDiskConnected(diskWithVBD(diskID, vmID, false), vmID, true, "Running")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("a running VM must mirror the runtime plug state (false), so a genuine disconnect surfaces as drift")
		}
	})

	t.Run("VM running, runtime plugged -> connected true (nominal, no drift)", func(t *testing.T) {
		got, err := resolveOpenIaaSDiskConnected(diskWithVBD(diskID, vmID, true), vmID, true, "Running")
		if err != nil || !got {
			t.Fatalf("want true,<nil> got %v,%v", got, err)
		}
	})

	t.Run("VM halted, runtime unplugged, intent connected -> preserve intent true (no power-off drift)", func(t *testing.T) {
		got, err := resolveOpenIaaSDiskConnected(diskWithVBD(diskID, vmID, false), vmID, true, "Halted")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("a halted VM must preserve the intent (true), not mirror the runtime unplug; kills the mirror-runtime-when-off mutant")
		}
	})

	t.Run("VM paused behaves like halted -> preserve intent true", func(t *testing.T) {
		got, err := resolveOpenIaaSDiskConnected(diskWithVBD(diskID, vmID, false), vmID, true, "Paused")
		if err != nil || !got {
			t.Fatalf("paused must preserve intent true, got %v,%v", got, err)
		}
	})

	t.Run("VM halted, intent disconnected -> preserve false", func(t *testing.T) {
		got, err := resolveOpenIaaSDiskConnected(diskWithVBD(diskID, vmID, false), vmID, false, "Halted")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("a halted VM with a disconnected intent must preserve false")
		}
	})

	t.Run("VBD absent for the configured VM -> attachment drift error, never a plain false", func(t *testing.T) {
		// The disk is readable but attached only to a DIFFERENT VM.
		disk := diskWithVBD(diskID, "other-vm", true)
		got, err := resolveOpenIaaSDiskConnected(disk, vmID, true, "Running")
		if err == nil {
			t.Fatal("a disk no longer attached to the configured VM must error (attachment drift), not be reported as merely disconnected")
		}
		if got {
			t.Fatalf("the error path must return the false zero-value, got %v", got)
		}
	})

	t.Run("unknown power state -> fail closed", func(t *testing.T) {
		_, err := resolveOpenIaaSDiskConnected(diskWithVBD(diskID, vmID, false), vmID, true, "Suspended")
		if err == nil {
			t.Fatal("an unknown power state must fail closed (cannot decide the connection state)")
		}
	})
}

// vdReadHandler routes a PRESENT disk read plus the VM power-state read used by
// the #350 override in openIaasVirtualDiskRead. The disk per-id GET returns
// diskBody (200); the VM per-id GET returns vmCode (+vmBody when 200). Any other
// call is a 500 (the Read must not mutate).
func vdReadHandler(diskBody string, vmCode int, vmBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/virtual_machines/"):
			if vmCode != http.StatusOK {
				w.WriteHeader(vmCode)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(vmBody))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/virtual_disks/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(diskBody))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

const vdPresentDiskBody = `{"id":"disk-x","virtualMachines":[{"id":"vm-1","connected":false}]}`

// TestOpenIaasVirtualDiskReadPowerOffPreservesIntent is the end-to-end wiring of
// the #350 fix: a halted VM reports the VBD unplugged, but Read must preserve the
// configured intent (connected=true) instead of writing the runtime false — so a
// stopped VM produces NO `connected: false -> true` drift.
func TestOpenIaasVirtualDiskReadPowerOffPreservesIntent(t *testing.T) {
	c := newAssignTestClient(t, vdReadHandler(vdPresentDiskBody, http.StatusOK, `{"id":"vm-1","powerState":"Halted"}`))
	d := diskTestData("disk-x", "vm-1")
	if err := d.Set("connected", true); err != nil { // the known intent
		t.Fatalf("seeding connected: %v", err)
	}
	if diags := openIaasVirtualDiskRead(context.Background(), d, c); diags.HasError() {
		t.Fatalf("a halted-VM disk read must succeed, got: %v", diags)
	}
	if !d.Get("connected").(bool) {
		t.Fatal("a halted VM must preserve connected=true (no power-off drift); the runtime unplug must not overwrite the intent")
	}
}

// TestOpenIaasVirtualDiskReadRunningMirrorsRuntime pins that while the VM runs, a
// genuine runtime disconnect IS reflected (so a real drift still surfaces).
func TestOpenIaasVirtualDiskReadRunningMirrorsRuntime(t *testing.T) {
	c := newAssignTestClient(t, vdReadHandler(vdPresentDiskBody, http.StatusOK, `{"id":"vm-1","powerState":"Running"}`))
	d := diskTestData("disk-x", "vm-1")
	if err := d.Set("connected", true); err != nil {
		t.Fatalf("seeding connected: %v", err)
	}
	if diags := openIaasVirtualDiskRead(context.Background(), d, c); diags.HasError() {
		t.Fatalf("a running-VM disk read must succeed, got: %v", diags)
	}
	if d.Get("connected").(bool) {
		t.Fatal("a running VM must mirror the runtime plug state (false): a genuine disconnect must surface as drift")
	}
}

// TestOpenIaasVirtualDiskReadFailsClosedWhenVMUnreadable pins the fail-closed
// guard: if the VM read maps to nil (since #384 a definitive 404), the provider
// cannot tell halted from running, so it must error rather than guess the
// connection state. (A genuine 403 errors at the client and is covered by
// TestOpenIaasVirtualDiskReadFailsClosedWhenVMReadErrors.)
func TestOpenIaasVirtualDiskReadFailsClosedWhenVMUnreadable(t *testing.T) {
	c := newAssignTestClient(t, vdReadHandler(vdPresentDiskBody, http.StatusNotFound, ``))
	d := diskTestData("disk-x", "vm-1")
	if err := d.Set("connected", true); err != nil {
		t.Fatalf("seeding connected: %v", err)
	}
	diags := openIaasVirtualDiskRead(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a VM read that maps to nil (404) must fail closed, never silently keep or guess the connection state")
	}
}

// TestOpenIaasVirtualDiskReadFailsClosedWhenVMReadErrors pins that a VM read that
// ERRORS (here a 400; since #384 a genuine 403 is also an error path, distinct
// from the 404 -> nil path) also fails closed — the provider never decides the
// connection state on a failed VM read.
func TestOpenIaasVirtualDiskReadFailsClosedWhenVMReadErrors(t *testing.T) {
	c := newAssignTestClient(t, vdReadHandler(vdPresentDiskBody, http.StatusBadRequest, ``))
	d := diskTestData("disk-x", "vm-1")
	if err := d.Set("connected", true); err != nil {
		t.Fatalf("seeding connected: %v", err)
	}
	if diags := openIaasVirtualDiskRead(context.Background(), d, c); !diags.HasError() {
		t.Fatal("a VM read error must fail closed, never decide the connection state")
	}
}

// TestOpenIaasVirtualDiskReadImportIdOnlySkipsVMRead pins that an id-only read
// (virtual_machine_id not yet known, e.g. right after import) SKIPS the override
// entirely and issues NO VM read — there is no intent/power state to resolve yet.
func TestOpenIaasVirtualDiskReadImportIdOnlySkipsVMRead(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/virtual_machines/"):
			t.Errorf("an id-only read must NOT read the VM, got: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/virtual_disks/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(vdPresentDiskBody))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	c := newAssignTestClient(t, handler)
	d := resourceOpenIaasVirtualDisk().TestResourceData()
	d.SetId("disk-x") // virtual_machine_id intentionally NOT set (id-only import)
	if diags := openIaasVirtualDiskRead(context.Background(), d, c); diags.HasError() {
		t.Fatalf("an id-only read must succeed without a VM read, got: %v", diags)
	}
}
