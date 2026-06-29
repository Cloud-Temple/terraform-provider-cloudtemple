package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// These unit tests pin the #386 fix: a VM CRUD operation that reads the virtual
// machine before acting must not panic when the client read returns (nil, nil).
// VirtualMachine().Read maps an HTTP 404 (absent) OR 403 (forbidden) to (nil,nil)
// via requireNotFoundOrOK(resp,403); the shared readVirtualMachineForOp guard
// converts that into an actionable diagnostic. Both status codes are exercised so
// the guard is proven identical before and after the API's 403->404 change.
//
// Mutation proof: removing the `if vm == nil` line in readVirtualMachineForOp
// makes TestReadVirtualMachineForOp's 404/403 cases return no diagnostic (RED) and
// makes TestComputeVirtualMachineDeleteNilReadDoesNotPanic panic at vm.PowerState
// (RED).

// TestReadVirtualMachineForOp covers the shared guard used by the three VM
// operation sites (update-customize, update-power, delete). The customize and
// power branches are gated (d.HasChange / the updatePower parameter) behind the
// full VM-update pipeline, so the guard logic is proven here directly rather than
// by driving those branches end-to-end.
func TestReadVirtualMachineForOp(t *testing.T) {
	t.Run("404 -> diagnostic, nil vm", func(t *testing.T) {
		c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
		vm, diags := readVirtualMachineForOp(context.Background(), c, "vm-1", "delete")
		if vm != nil {
			t.Fatalf("expected nil vm on a 404 read, got %+v", vm)
		}
		if !diags.HasError() {
			t.Fatal("expected an error diagnostic on a 404 read, got none")
		}
		diagsContain(t, diags, "could not be read")
	})
	t.Run("403 -> diagnostic, nil vm", func(t *testing.T) {
		c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) })
		vm, diags := readVirtualMachineForOp(context.Background(), c, "vm-1", "update")
		if vm != nil {
			t.Fatalf("expected nil vm on a 403 read, got %+v", vm)
		}
		if !diags.HasError() {
			t.Fatal("expected an error diagnostic on a 403 read, got none")
		}
		diagsContain(t, diags, "could not be read")
	})
	t.Run("500 -> diagnostic (read error)", func(t *testing.T) {
		c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) })
		vm, diags := readVirtualMachineForOp(context.Background(), c, "vm-1", "power")
		if vm != nil {
			t.Fatalf("expected nil vm on a 500 read, got %+v", vm)
		}
		if !diags.HasError() {
			t.Fatal("expected an error diagnostic on a 500 read, got none")
		}
	})
	t.Run("200 -> vm, no diagnostic", func(t *testing.T) {
		c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"vm-1"}`))
		})
		vm, diags := readVirtualMachineForOp(context.Background(), c, "vm-1", "update")
		if diags.HasError() {
			t.Fatalf("a valid 200 read must not error, got: %v", diags)
		}
		if vm == nil || vm.ID != "vm-1" {
			t.Fatalf("expected a non-nil vm with id %q, got %+v", "vm-1", vm)
		}
	})
}

// TestComputeVirtualMachineDeleteNilReadDoesNotPanic: destroy reads the VM first
// (unconditionally) to power it off before deleting. An absent/forbidden (nil,nil)
// read must fail closed with a diagnostic, never panic at vm.PowerState. Fail
// closed (not idempotent): nil is ambiguous between 404 and 403, so concluding
// "already deleted" needs a definitive 404 — that distinction is #384.
func TestComputeVirtualMachineDeleteNilReadDoesNotPanic(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) })
			d := resourceVirtualMachine().TestResourceData()
			d.SetId("vm-1")

			diags := computeVirtualMachineDelete(context.Background(), d, c)
			if !diags.HasError() {
				t.Fatalf("a %d read on delete must return a diagnostic, got none", code)
			}
			diagsContain(t, diags, "could not be read")
		})
	}
}
