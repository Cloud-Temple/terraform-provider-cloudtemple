package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// adapterTestData builds the resource data of a standalone OpenIaaS network
// adapter bound to vmID.
func adapterTestData(id, vmID string) *schema.ResourceData {
	d := resourceOpenIaasNetworkAdapter().TestResourceData()
	d.SetId(id)
	_ = d.Set("virtual_machine_id", vmID)
	return d
}

// TestResolveOpenIaaSAdapterAttached pins the power-off drift fix (#537):
// `attached` mirrors the runtime VIF plug state ONLY while the VM runs; on a
// halted/paused VM (where the platform reports every VIF unplugged) it preserves
// the known intent; an unknown power state fails closed.
func TestResolveOpenIaaSAdapterAttached(t *testing.T) {
	const adapterID, vmID = "vif-1", "vm-1"

	t.Run("running VM -> a genuine unplug surfaces as drift", func(t *testing.T) {
		got, err := resolveOpenIaaSAdapterAttached(false, true, adapterID, vmID, "Running")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("a running VM must mirror the runtime: an unplugged VIF must be reported as attached=false")
		}
	})

	t.Run("running VM -> a plugged VIF stays true", func(t *testing.T) {
		got, err := resolveOpenIaaSAdapterAttached(true, true, adapterID, vmID, "Running")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("a running VM with a plugged VIF must report attached=true")
		}
	})

	// The regression itself: this is the exact live shape (#537) — halted VM, the
	// platform reports the VIF unplugged, the user's intent is the schema default.
	t.Run("halted VM -> preserves the attached intent (no power-off drift)", func(t *testing.T) {
		got, err := resolveOpenIaaSAdapterAttached(false, true, adapterID, vmID, "Halted")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("a halted VM must preserve attached=true: the runtime unplug is not the user's intent")
		}
	})

	t.Run("paused VM -> preserves the attached intent", func(t *testing.T) {
		got, err := resolveOpenIaaSAdapterAttached(false, true, adapterID, vmID, "Paused")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Fatal("a paused VM must preserve attached=true like a halted one")
		}
	})

	// Symmetry guard: preserving the intent must work BOTH ways, otherwise the fix
	// would silently re-attach a deliberately detached adapter on every read.
	t.Run("halted VM -> preserves a detached intent", func(t *testing.T) {
		got, err := resolveOpenIaaSAdapterAttached(false, false, adapterID, vmID, "Halted")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("a halted VM with a detached intent must preserve false, never flip to true")
		}
	})

	// A halted VM reporting a PLUGGED VIF is not a state the platform produces,
	// but the decision must still follow the intent rather than the runtime:
	// this pins that "off" reads priorAttached and nothing else.
	t.Run("halted VM -> the intent wins over the runtime", func(t *testing.T) {
		got, err := resolveOpenIaaSAdapterAttached(true, false, adapterID, vmID, "Halted")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Fatal("on a halted VM the decision must come from the intent (false), not from the runtime")
		}
	})

	t.Run("unknown power state -> fail closed", func(t *testing.T) {
		got, err := resolveOpenIaaSAdapterAttached(true, true, adapterID, vmID, "Suspended")
		if err == nil {
			t.Fatal("an unknown power state must fail closed (cannot decide the attachment state)")
		}
		if got {
			t.Fatalf("the error path must return the false zero-value, got %v", got)
		}
	})
}

// TestOpenIaasAttachRequiresRunningVM pins the second half of #537: an explicit
// attach against a halted VM is refused UP FRONT with an actionable message,
// instead of being sent and failing as an opaque platform activity.
func TestOpenIaasAttachRequiresRunningVM(t *testing.T) {
	const adapterID, vmID = "vif-1", "vm-1"

	t.Run("running VM -> attach is allowed", func(t *testing.T) {
		if err := openIaasAttachRequiresRunningVM(adapterID, vmID, "Running"); err != nil {
			t.Fatalf("attaching on a running VM must be allowed, got: %v", err)
		}
	})

	for _, powerState := range []string{"Halted", "Paused"} {
		t.Run("powered-off VM ("+powerState+") -> refused with an actionable message", func(t *testing.T) {
			err := openIaasAttachRequiresRunningVM(adapterID, vmID, powerState)
			if err == nil {
				t.Fatal("attaching a VIF on a non-running VM must be refused before the API call")
			}
			// The whole point of the guard is that the user learns what to do; a
			// bare "refused" would just replace one opaque failure with another.
			for _, want := range []string{adapterID, vmID, "attached = false"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("the diagnostic must mention %q so the user can act on it, got: %v", want, err)
				}
			}
		})
	}

	t.Run("unknown power state -> fail closed", func(t *testing.T) {
		if err := openIaasAttachRequiresRunningVM(adapterID, vmID, "Suspended"); err == nil {
			t.Fatal("an unknown power state must fail closed, never attempt the attach")
		}
	})
}

// vifReadHandler routes a PRESENT adapter read plus the VM power-state read used
// by the #537 override in openIaasNetworkAdapterRead. Any other call is a 500:
// the Read must not mutate anything.
func vifReadHandler(adapterBody string, vmCode int, vmBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/virtual_machines/"):
			if vmCode != http.StatusOK {
				w.WriteHeader(vmCode)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(vmBody))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/network_adapters/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(adapterBody))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// The live shape captured on DEV for #537: a VIF of a halted VM, reported
// unplugged by the platform.
const vifUnpluggedBody = `{"id":"vif-x","virtualMachineId":"vm-1","macAddress":"b2:6a:ee:79:d4:49","attached":false,"txChecksumming":true,"network":{"id":"net-1"}}`

// TestOpenIaasNetworkAdapterReadPowerOffPreservesIntent is the end-to-end wiring
// of the #537 fix: a halted VM reports the VIF unplugged, but Read must preserve
// the configured intent (attached=true) instead of writing the runtime false —
// so adding an adapter to a stopped VM produces NO `attached: false -> true`
// drift, and therefore no apply that the platform refuses.
func TestOpenIaasNetworkAdapterReadPowerOffPreservesIntent(t *testing.T) {
	c := newAssignTestClient(t, vifReadHandler(vifUnpluggedBody, http.StatusOK, `{"id":"vm-1","powerState":"Halted"}`))
	d := adapterTestData("vif-x", "vm-1")
	if err := d.Set("attached", true); err != nil { // the known intent (schema default)
		t.Fatalf("seeding attached: %v", err)
	}
	if diags := openIaasNetworkAdapterRead(context.Background(), d, c); diags.HasError() {
		t.Fatalf("a halted-VM adapter read must succeed, got: %v", diags)
	}
	if !d.Get("attached").(bool) {
		t.Fatal("a halted VM must preserve attached=true (no power-off drift); the runtime unplug must not overwrite the intent")
	}
}

// TestOpenIaasNetworkAdapterReadRunningMirrorsRuntime pins that while the VM
// runs, a genuine unplug IS reflected — the fix must not blind the provider to a
// real drift.
func TestOpenIaasNetworkAdapterReadRunningMirrorsRuntime(t *testing.T) {
	c := newAssignTestClient(t, vifReadHandler(vifUnpluggedBody, http.StatusOK, `{"id":"vm-1","powerState":"Running"}`))
	d := adapterTestData("vif-x", "vm-1")
	if err := d.Set("attached", true); err != nil {
		t.Fatalf("seeding attached: %v", err)
	}
	if diags := openIaasNetworkAdapterRead(context.Background(), d, c); diags.HasError() {
		t.Fatalf("a running-VM adapter read must succeed, got: %v", diags)
	}
	if d.Get("attached").(bool) {
		t.Fatal("a running VM must mirror the runtime plug state (false): a genuine unplug must surface as drift")
	}
}

// TestOpenIaasNetworkAdapterReadFailsClosedWhenVMUnreadable pins the fail-closed
// guard: a VM read mapping to nil (since #384 a definitive 404) leaves the
// provider unable to tell halted from running, so it must error rather than
// guess the attachment state.
func TestOpenIaasNetworkAdapterReadFailsClosedWhenVMUnreadable(t *testing.T) {
	c := newAssignTestClient(t, vifReadHandler(vifUnpluggedBody, http.StatusNotFound, ``))
	d := adapterTestData("vif-x", "vm-1")
	if err := d.Set("attached", true); err != nil {
		t.Fatalf("seeding attached: %v", err)
	}
	if diags := openIaasNetworkAdapterRead(context.Background(), d, c); !diags.HasError() {
		t.Fatal("a VM read that maps to nil (404) must fail closed, never guess the attachment state")
	}
}

// TestOpenIaasNetworkAdapterReadFailsClosedWhenVMReadErrors pins that a VM read
// that ERRORS (here a 400; since #384 a genuine 403 is also an error path,
// distinct from the 404 -> nil path) also fails closed.
func TestOpenIaasNetworkAdapterReadFailsClosedWhenVMReadErrors(t *testing.T) {
	c := newAssignTestClient(t, vifReadHandler(vifUnpluggedBody, http.StatusBadRequest, ``))
	d := adapterTestData("vif-x", "vm-1")
	if err := d.Set("attached", true); err != nil {
		t.Fatalf("seeding attached: %v", err)
	}
	if diags := openIaasNetworkAdapterRead(context.Background(), d, c); !diags.HasError() {
		t.Fatal("a VM read error must fail closed, never decide the attachment state")
	}
}

// TestOpenIaasNetworkAdapterReadUnboundSkipsVMRead pins that an adapter the API
// reports with no owning VM issues NO VM read and keeps the flattened runtime
// value: there is no power state to resolve it against.
func TestOpenIaasNetworkAdapterReadUnboundSkipsVMRead(t *testing.T) {
	const unbound = `{"id":"vif-x","virtualMachineId":"","macAddress":"b2:6a:ee:79:d4:49","attached":false,"network":{"id":"net-1"}}`
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/virtual_machines/"):
			t.Errorf("an adapter with no owning VM must NOT read the VM, got: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/network_adapters/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(unbound))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	c := newAssignTestClient(t, handler)
	d := resourceOpenIaasNetworkAdapter().TestResourceData()
	d.SetId("vif-x")
	if diags := openIaasNetworkAdapterRead(context.Background(), d, c); diags.HasError() {
		t.Fatalf("an unbound adapter read must succeed without a VM read, got: %v", diags)
	}
}

// TestOpenIaasNetworkAdapterUpdateRefusesAttachOnHaltedVM is the decisive guard
// of the second half of #537: the doomed `connect` must NEVER reach the wire.
// Before the fix the provider issued it and surfaced the platform's opaque
// activity failure ("VM state is halted but should be running") as a bare UUID.
func TestOpenIaasNetworkAdapterUpdateRefusesAttachOnHaltedVM(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/connect") {
			t.Errorf("the connect request must never be sent for a halted VM, got: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/virtual_machines/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"vm-1","powerState":"Halted"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/network_adapters/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(vifUnpluggedBody))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	c := newAssignTestClient(t, handler)

	// State: the adapter is currently detached; the configuration now asks for it
	// to be attached — the exact transition the platform refuses on a halted VM.
	// Built through a real Diff so d.HasChange("attached") is genuinely true:
	// a plain TestResourceData reports no change and would skip the guarded block
	// entirely, making this test pass for the wrong reason.
	res := resourceOpenIaasNetworkAdapter()
	prior := &terraform.InstanceState{ID: "vif-x", Attributes: map[string]string{
		"id":                 "vif-x",
		"virtual_machine_id": "vm-1",
		"network_id":         "net-1",
		"attached":           "false",
	}}
	cfg := terraform.NewResourceConfigRaw(map[string]interface{}{
		"virtual_machine_id": "vm-1",
		"network_id":         "net-1",
		"attached":           true,
	})
	diff, err := res.Diff(context.Background(), prior, cfg, nil)
	if err != nil {
		t.Fatalf("building the diff: %v", err)
	}
	d, err := schema.InternalMap(res.Schema).Data(prior, diff)
	if err != nil {
		t.Fatalf("building the resource data: %v", err)
	}
	if !d.HasChange("attached") {
		t.Fatal("test setup is wrong: attached must be a real change, otherwise the guard is never reached")
	}

	diags := openIaasNetworkAdapterUpdate(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("attaching an adapter while its VM is halted must fail with an explicit diagnostic")
	}
	if !strings.Contains(diags[0].Summary, "attached = false") {
		t.Fatalf("the diagnostic must tell the user how to proceed, got: %s", diags[0].Summary)
	}
}
