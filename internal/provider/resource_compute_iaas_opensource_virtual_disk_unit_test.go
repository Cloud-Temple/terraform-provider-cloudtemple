package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// diagsContain fails the test unless some diagnostic's Summary contains substr.
// The verdict tests use it to pin WHICH branch produced the error: a regression
// misclassifying deviceExistsOutOfScope as deviceStillInScope (or vice-versa)
// would otherwise pass, since both fail closed with an error.
func diagsContain(t *testing.T, diags diag.Diagnostics, substr string) {
	t.Helper()
	for _, dg := range diags {
		if strings.Contains(dg.Summary, substr) {
			return
		}
	}
	t.Fatalf("expected a diagnostic whose summary contains %q, got %v", substr, diags)
}

// diskTestData builds a *schema.ResourceData for the virtual_disk resource with
// the id and virtual_machine_id pre-set, so the CRUD functions resolve d.Id()
// and the VM-scoped strict listing exactly as Terraform would at runtime.
func diskTestData(id, vmID string) *schema.ResourceData {
	d := resourceOpenIaasVirtualDisk().TestResourceData()
	d.SetId(id)
	_ = d.Set("virtual_machine_id", vmID)
	return d
}

// vdMutationGuard is a tiny accounting struct so the tests can assert, beyond
// "no panic", that the dangerous mutation/disconnect/delete endpoints are NEVER
// hit on the nil-read paths.
type vdMutationGuard struct {
	t        *testing.T
	mutating int
}

func (g *vdMutationGuard) flag(w http.ResponseWriter, r *http.Request) {
	g.mutating++
	g.t.Errorf("unexpected mutating call on a nil-read path: %s %s", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusInternalServerError)
}

// vdHandler routes the OpenIaaS virtual disk endpoints for these unit tests:
//   - GET .../virtual_disks/{id}  -> 403, which the client maps to a nil read
//     (requireNotFoundOrOK(resp, 403)); this is the exact crash trigger of #325.
//   - GET .../virtual_disks       -> a strict listing; scoped (?virtualMachineId)
//     vs tenant-wide is distinguished by the filter query param.
//
// Any connect/disconnect/attach/detach/update/relocate/DELETE call is treated as
// a forbidden mutation on the nil path and fails the test loudly.
func vdHandler(g *vdMutationGuard, scopedBody, tenantBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		// Per-id read: a trailing id segment after /virtual_disks/.
		case r.Method == http.MethodGet && strings.Contains(path, "/virtual_disks/") &&
			!strings.HasSuffix(path, "/connect") && !strings.HasSuffix(path, "/disconnect") &&
			!strings.HasSuffix(path, "/attach") && !strings.HasSuffix(path, "/detach") &&
			!strings.HasSuffix(path, "/relocate"):
			w.WriteHeader(http.StatusForbidden)
		// Strict listing (no id segment): /virtual_disks exactly.
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/virtual_disks"):
			body := tenantBody
			if r.URL.Query().Get("virtualMachineId") != "" {
				body = scopedBody
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		default:
			g.flag(w, r)
		}
	}
}

// TestOpenIaasVirtualDiskReadNilConfirmedDrops pins that the refactored Read
// (now delegating to confirmOpenIaaSVirtualDiskDeleted) still DROPS the state on
// a deletion proven by both strict listings being empty.
//
// Mutation proof: if confirmOpenIaaSVirtualDiskDeleted ever returned a non-empty
// scoped/tenant set here, the verdict would be deviceStillInScope/OutOfScope and
// the function would error instead of clearing the id — this test goes RED.
func TestOpenIaasVirtualDiskReadNilConfirmedDrops(t *testing.T) {
	g := &vdMutationGuard{t: t}
	c := newAssignTestClient(t, vdHandler(g, `[]`, `[]`))
	d := diskTestData("disk-x", "vm-1")

	if diags := openIaasVirtualDiskRead(context.Background(), d, c); diags.HasError() {
		t.Fatalf("a deletion proven by empty strict listings must not error, got: %v", diags)
	}
	if d.Id() != "" {
		t.Fatalf("a confirmed deletion must clear the id, got %q", d.Id())
	}
}

// TestOpenIaasVirtualDiskReadNilStillInScopeRefusesToDrop pins the never-drop
// half: a per-id read that returns nil while the disk is STILL in the scoped
// listing (access restriction / API inconsistency) must fail closed and keep
// the id in state.
func TestOpenIaasVirtualDiskReadNilStillInScopeRefusesToDrop(t *testing.T) {
	g := &vdMutationGuard{t: t}
	c := newAssignTestClient(t, vdHandler(g, `[{"id":"disk-x"}]`, `[{"id":"disk-x"}]`))
	d := diskTestData("disk-x", "vm-1")

	diags := openIaasVirtualDiskRead(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a disk still listed on the VM must NOT be dropped: expected an error")
	}
	diagsContain(t, diags, "still listed on virtual machine")
	if d.Id() == "" {
		t.Fatal("refusing to drop must keep the id in state, but it was cleared")
	}
}

// TestOpenIaasVirtualDiskUpdateNilRefusesToMutate pins the #325 fix on the Update
// path: a nil pre-update read must yield a clear error and NEVER reach the
// HasChange-gated loops that dereference disk.VirtualMachines.
//
// Mutation proof (verified): removing the `if disk == nil` guard makes this test
// go RED — but NOT via panic here. TestResourceData carries no diff, so every
// d.HasChange is false and the dangerous loops (alreadyAttached / connected VMs)
// are skipped; Update then falls through to the trailing Read, which on a
// confirmed deletion clears the id and returns NO error, so the "expected an
// error" assertion fails. In production an Update runs under a real diff: a
// name/size or VM-attachment (virtual_machine_id/bootable/mode) change opens a
// HasChange branch that ranges over disk.VirtualMachines, so the unguarded path
// would dereference the nil disk and panic — the crash this guard prevents. The
// mutation guard also fails the test if any connect/attach/etc. call is
// attempted on this path.
func TestOpenIaasVirtualDiskUpdateNilRefusesToMutate(t *testing.T) {
	g := &vdMutationGuard{t: t}
	c := newAssignTestClient(t, vdHandler(g, `[]`, `[]`))
	d := diskTestData("disk-x", "vm-1")

	diags := openIaasVirtualDiskUpdate(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a nil pre-update read must refuse to mutate and return an error")
	}
	if g.mutating != 0 {
		t.Fatalf("update on a nil read must issue zero mutating calls, got %d", g.mutating)
	}
}

// TestOpenIaasVirtualDiskDeleteNilConfirmedIsIdempotent pins that a destroy of a
// disk whose deletion is proven (both strict listings empty) succeeds as a no-op
// WITHOUT issuing any disconnect/delete call, and without panicking.
//
// Mutation proof: remove the `if disk == nil` guard and the disconnect loop
// dereferences a nil disk -> panic. Make the guard return nil unconditionally
// (skipping the verdict switch) and TestOpenIaasVirtualDiskDeleteNilStillInScope
// below goes RED.
func TestOpenIaasVirtualDiskDeleteNilConfirmedIsIdempotent(t *testing.T) {
	g := &vdMutationGuard{t: t}
	c := newAssignTestClient(t, vdHandler(g, `[]`, `[]`))
	d := diskTestData("disk-x", "vm-1")

	if diags := openIaasVirtualDiskDelete(context.Background(), d, c); diags.HasError() {
		t.Fatalf("a destroy with a confirmed deletion must be a no-op success, got: %v", diags)
	}
	if g.mutating != 0 {
		t.Fatalf("delete on a confirmed-gone disk must issue zero disconnect/delete calls, got %d", g.mutating)
	}
}

// TestOpenIaasVirtualDiskDeleteNilStillInScopeErrors pins the never-orphan half
// of the Delete fix: a nil read with the disk STILL listed on the VM must refuse
// to assume a deletion (it could be a forbidden-but-existing disk) and error,
// rather than silently dropping it from state.
func TestOpenIaasVirtualDiskDeleteNilStillInScopeErrors(t *testing.T) {
	g := &vdMutationGuard{t: t}
	c := newAssignTestClient(t, vdHandler(g, `[{"id":"disk-x"}]`, `[{"id":"disk-x"}]`))
	d := diskTestData("disk-x", "vm-1")

	diags := openIaasVirtualDiskDelete(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a still-listed disk must not be assumed deleted: expected an error")
	}
	diagsContain(t, diags, "refusing to assume it was deleted")
	if g.mutating != 0 {
		t.Fatalf("delete must not issue any disconnect/delete call when refusing, got %d", g.mutating)
	}
}

// TestOpenIaasVirtualDiskReadNilOutOfScopeRefusesToDrop pins the third verdict on
// Read: a nil per-id read with the disk ABSENT from the VM-scoped listing but
// still present tenant-wide is drift (detached or moved), never a deletion — the
// id must be kept in state and an actionable error returned.
func TestOpenIaasVirtualDiskReadNilOutOfScopeRefusesToDrop(t *testing.T) {
	g := &vdMutationGuard{t: t}
	// scoped (?virtualMachineId) empty, tenant-wide still lists the disk.
	c := newAssignTestClient(t, vdHandler(g, `[]`, `[{"id":"disk-x"}]`))
	d := diskTestData("disk-x", "vm-1")

	diags := openIaasVirtualDiskRead(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a disk detached/moved (out of scope) is drift, not a deletion: expected an error")
	}
	// Pin the exact verdict: a misclassification to still-in-scope also fails
	// closed and would otherwise pass this test.
	diagsContain(t, diags, "still exists platform-side")
	if d.Id() == "" {
		t.Fatal("drift must keep the id in state, but it was cleared")
	}
}

// TestOpenIaasVirtualDiskDeleteNilOutOfScopeErrors pins the third verdict on
// Delete: a disk absent from the VM scope but still present tenant-wide must not
// be assumed deleted (it would orphan a platform-side disk) — error, no mutation.
func TestOpenIaasVirtualDiskDeleteNilOutOfScopeErrors(t *testing.T) {
	g := &vdMutationGuard{t: t}
	c := newAssignTestClient(t, vdHandler(g, `[]`, `[{"id":"disk-x"}]`))
	d := diskTestData("disk-x", "vm-1")

	diags := openIaasVirtualDiskDelete(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a disk still present tenant-wide must not be assumed deleted: expected an error")
	}
	diagsContain(t, diags, "still exists platform-side")
	if g.mutating != 0 {
		t.Fatalf("delete must not issue any disconnect/delete call when refusing, got %d", g.mutating)
	}
}

// TestOpenIaasVirtualDiskDeleteListingErrorFailsClosed pins the fail-closed
// contract of confirmOpenIaaSVirtualDiskDeleted: if a strict listing itself
// fails (here a 500 on the scoped listing), the deletion can NEVER be confirmed,
// so Delete must error rather than assume the disk is gone — and issue no
// disconnect/delete call.
func TestOpenIaasVirtualDiskDeleteListingErrorFailsClosed(t *testing.T) {
	g := &vdMutationGuard{t: t}
	c := newAssignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Strict listing fails: an absence can never be proven.
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/virtual_disks"):
			w.WriteHeader(http.StatusInternalServerError)
		// Per-id read: 403 -> nil, the crash trigger.
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/virtual_disks/"):
			w.WriteHeader(http.StatusForbidden)
		default:
			g.flag(w, r)
		}
	})
	d := diskTestData("disk-x", "vm-1")

	diags := openIaasVirtualDiskDelete(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a destroy whose deletion cannot be confirmed (listing error) must fail closed")
	}
	if g.mutating != 0 {
		t.Fatalf("delete must issue no disconnect/delete call when failing closed, got %d", g.mutating)
	}
}
