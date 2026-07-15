package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// --- recording fake seams (offline, no client) — VMware siblings of the OpenIaaS
// fakes in cycle_compute_lifecycle_test.go (distinct names, same package). ---

type fakeVMwareVMSeam struct {
	log       *[]string
	findID    string
	findErr   error
	delErr    error // when non-nil, DeleteAndWait fails (e.g. a VMware 403-on-absent)
	exists    bool  // Exists() result, used to confirm a 403 delete
	existsErr error
}

func (f fakeVMwareVMSeam) DeleteAndWait(_ context.Context, id string) error {
	*f.log = append(*f.log, "vm.delete:"+id)
	return f.delErr
}
func (f fakeVMwareVMSeam) Exists(_ context.Context, id string) (bool, error) {
	*f.log = append(*f.log, "vm.exists:"+id)
	return f.exists, f.existsErr
}
func (f fakeVMwareVMSeam) PowerOffAndWait(_ context.Context, id string) error {
	*f.log = append(*f.log, "vm.poweroff:"+id)
	return nil
}
func (f fakeVMwareVMSeam) FindIDByName(_ context.Context, name, dcID string) (string, error) {
	*f.log = append(*f.log, "vm.find:"+name+"@"+dcID)
	return f.findID, f.findErr
}

type fakeVMwareDiskSeam struct {
	log       *[]string
	ids       []string // FindIDsByVM returns these
	delErr    error    // when non-nil, DeleteAndWait fails (e.g. a 409)
	exists    bool     // Exists() result
	existsErr error
}

func (f fakeVMwareDiskSeam) FindIDsByVM(_ context.Context, vmID string) ([]string, error) {
	*f.log = append(*f.log, "disk.find:"+vmID)
	return f.ids, nil
}
func (f fakeVMwareDiskSeam) DeleteAndWait(_ context.Context, id string) error {
	*f.log = append(*f.log, "disk.delete:"+id)
	return f.delErr
}
func (f fakeVMwareDiskSeam) Exists(_ context.Context, id string) (bool, error) {
	*f.log = append(*f.log, "disk.exists:"+id)
	return f.exists, f.existsErr
}

type fakeVMwareAdapterSeam struct {
	log       *[]string
	ids       []string // FindIDsByVM returns these
	delErr    error    // when non-nil, DeleteAndWait fails
	exists    bool     // Exists() result
	existsErr error
}

func (f fakeVMwareAdapterSeam) FindIDsByVM(_ context.Context, vmID string) ([]string, error) {
	*f.log = append(*f.log, "adapter.find:"+vmID)
	return f.ids, nil
}
func (f fakeVMwareAdapterSeam) DeleteAndWait(_ context.Context, id string) error {
	*f.log = append(*f.log, "adapter.delete:"+id)
	return f.delErr
}
func (f fakeVMwareAdapterSeam) Exists(_ context.Context, id string) (bool, error) {
	*f.log = append(*f.log, "adapter.exists:"+id)
	return f.exists, f.existsErr
}

// TestConfirmComputeDeleteErr pins the VMware compute delete decision: 404 →
// idempotent success; a 403 is CONFIRMED via a by-id existence re-check and
// accepted ONLY when the id is proven absent (the false-"orphan" fix), surfaced
// when it still exists, and failed-closed when the re-check itself errors; other
// errors surface unchanged. Mutations: treat 403 as always-success → the
// still-present case wrongly passes → RED; treat 403 as always-failure → the
// absent case fails → RED.
func TestConfirmComputeDeleteErr(t *testing.T) {
	forbidden := client.StatusError{Code: http.StatusForbidden, Body: "Forbidden."}
	notFound := client.StatusError{Code: http.StatusNotFound}
	serverErr := client.StatusError{Code: http.StatusInternalServerError}
	absent := func() (bool, error) { return false, nil }
	present := func() (bool, error) { return true, nil }
	recheckFailed := func() (bool, error) { return false, errors.New("transport error: cannot confirm") }

	if confirmComputeDeleteErr(nil, present, "id-x") != nil {
		t.Fatal("nil error must be success")
	}
	if confirmComputeDeleteErr(notFound, present, "id-x") != nil {
		t.Fatal("a 404 must be idempotent success without re-checking existence")
	}
	if err := confirmComputeDeleteErr(forbidden, absent, "id-x"); err != nil {
		t.Fatalf("a 403 with the id confirmed ABSENT must be success (no false orphan), got %v", err)
	}
	if confirmComputeDeleteErr(forbidden, present, "id-x") == nil {
		t.Fatal("a 403 with the id STILL existing must surface as a real failure")
	}
	if confirmComputeDeleteErr(forbidden, recheckFailed, "id-x") == nil {
		t.Fatal("a 403 whose absence cannot be confirmed must fail closed")
	}
	if confirmComputeDeleteErr(serverErr, absent, "id-x") == nil {
		t.Fatal("a 5xx must surface unchanged (never silently accepted)")
	}
}

// TestVMwareVMTeardownConfirms403AsAbsent pins the wiring: when the deferred VM
// teardown re-deletes an already-gone VM (VMware answers 403, not 404) and the
// by-id existence re-check shows it absent, the teardown is a clean SUCCESS — not
// a false "possible orphan". If the re-check still shows it present, it is a real
// failure. A separate case pins fail-closed when the re-check itself errors.
func TestVMwareVMTeardownConfirms403AsAbsent(t *testing.T) {
	forbidden := client.StatusError{Code: http.StatusForbidden, Body: "Forbidden."}

	// VM already gone: delete 403s, the by-id existence re-check says absent → ok.
	var log []string
	cl := NewCleanup()
	registerVMwareVMTeardown(cl, fakeVMwareVMSeam{log: &log, delErr: forbidden, exists: false},
		&vmwareVMTeardownRef{Name: "vm-x", DatacenterID: "dc-1", ID: "vm-id", Resolved: true})
	if f := cl.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("a 403 on an already-gone VM must be confirmed absent (no false orphan), got failures %v", f)
	}

	// VM still present: delete 403s, the re-check says it still exists → real failure.
	var log2 []string
	cl2 := NewCleanup()
	registerVMwareVMTeardown(cl2, fakeVMwareVMSeam{log: &log2, delErr: forbidden, exists: true},
		&vmwareVMTeardownRef{Name: "vm-x", DatacenterID: "dc-1", ID: "vm-id", Resolved: true})
	if f := cl2.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("a 403 with the VM STILL present must be a real teardown failure, got %v", f)
	}

	// Re-check ITSELF errors (e.g. a 5xx on Read): unknown existence → FAIL CLOSED
	// (a teardown failure), never silently accepted.
	var log3 []string
	cl3 := NewCleanup()
	registerVMwareVMTeardown(cl3, fakeVMwareVMSeam{log: &log3, delErr: forbidden, existsErr: errors.New("503 on read")},
		&vmwareVMTeardownRef{Name: "vm-x", DatacenterID: "dc-1", ID: "vm-id", Resolved: true})
	if f := cl3.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("a 403 whose existence re-check errors must fail closed (teardown failure), got %v", f)
	}
}

// TestVMwareLeafTeardownConfirms403WhenParentVMGone pins Codex's exact concern: a
// deferred LEAF (disk/adapter) re-delete after the explicit deprovision already
// removed the parent VM gets a 403; the confirm is a BY-ID existence re-check
// (Exists), independent of the now-gone VM, so an absent leaf is a clean success
// (not a false orphan), and a still-present leaf is a real failure.
func TestVMwareLeafTeardownConfirms403WhenParentVMGone(t *testing.T) {
	forbidden := client.StatusError{Code: http.StatusForbidden, Body: "Forbidden."}

	// Disk: resolved, delete 403s, Exists=false (gone with its VM) → success.
	var dl []string
	dcl := NewCleanup()
	registerVMwareDiskTeardown(dcl, fakeVMwareDiskSeam{log: &dl, delErr: forbidden, exists: false},
		&vmwareDiskTeardownRef{VMID: "vm-gone", ID: "disk-1", Resolved: true})
	if f := dcl.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("a 403 disk delete confirmed absent by-id must succeed even when the VM is gone, got %v", f)
	}

	// Adapter: resolved, delete 403s, Exists=true (still there) → real failure.
	var al []string
	acl := NewCleanup()
	registerVMwareNetworkAdapterTeardown(acl, fakeVMwareAdapterSeam{log: &al, delErr: forbidden, exists: true},
		&vmwareAdapterTeardownRef{VMID: "vm-x", ID: "ad-1", Resolved: true})
	if f := acl.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("a 403 adapter delete with the adapter still present must be a real failure, got %v", f)
	}
}

// TestRegisterVMwareVMTeardownResolvedDeletesByID: a resolved ref deletes by id
// (power-off best-effort first), never falling back to find-by-name. Mutation:
// drop the `ref.Resolved` check → find-by-name is used → "vm.find" appears.
func TestRegisterVMwareVMTeardownResolvedDeletesByID(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareVMTeardown(cl, fakeVMwareVMSeam{log: &log, findID: "should-not-be-used"},
		&vmwareVMTeardownRef{Name: "vm-a", DatacenterID: "dc-1", ID: "vm-id-1", Resolved: true})
	cl.TeardownAll(context.Background())
	if indexOf(log, "vm.find") != -1 {
		t.Fatalf("a resolved ref must delete by id, not find by name; log=%v", log)
	}
	if indexOf(log, "vm.delete:vm-id-1") == -1 {
		t.Fatalf("must delete by the resolved id; log=%v", log)
	}
}

// TestRegisterVMwareVMTeardownUnresolvedFindsByName: an UNRESOLVED ref (create id
// never came back) finds the VM by its deterministic name within the datacenter
// and deletes it — the reason the teardown is registered BEFORE the create.
func TestRegisterVMwareVMTeardownUnresolvedFindsByName(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareVMTeardown(cl, fakeVMwareVMSeam{log: &log, findID: "found-vm-9"},
		&vmwareVMTeardownRef{Name: "vm-b", DatacenterID: "dc-1"}) // not resolved
	cl.TeardownAll(context.Background())
	if indexOf(log, "vm.find:vm-b@dc-1") == -1 || indexOf(log, "vm.delete:found-vm-9") == -1 {
		t.Fatalf("unresolved ref must find-by-name then delete; log=%v", log)
	}
}

// TestRegisterVMwareVMTeardownNotFoundIsNoop: unresolved + not found → nothing to
// do, no delete (idempotent: never created / already gone).
func TestRegisterVMwareVMTeardownNotFoundIsNoop(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareVMTeardown(cl, fakeVMwareVMSeam{log: &log, findID: ""},
		&vmwareVMTeardownRef{Name: "vm-c", DatacenterID: "dc-1"})
	if f := cl.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("not-found teardown must be a no-op success, got failures %v", f)
	}
	if indexOf(log, "vm.delete") != -1 {
		t.Fatalf("nothing must be deleted when not found; log=%v", log)
	}
}

// TestRegisterVMwareDiskTeardownResolvedDeletesByID: a resolved disk deletes by
// id, never listing the VM. Mutation: drop the Resolved check → "disk.find"
// appears.
func TestRegisterVMwareDiskTeardownResolvedDeletesByID(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareDiskTeardown(cl, fakeVMwareDiskSeam{log: &log, ids: []string{"should-not-be-used"}},
		&vmwareDiskTeardownRef{VMID: "vm-1", ID: "disk-7", Resolved: true})
	cl.TeardownAll(context.Background())
	if indexOf(log, "disk.find") != -1 {
		t.Fatalf("a resolved disk must delete by id, not list the VM; log=%v", log)
	}
	if indexOf(log, "disk.delete:disk-7") == -1 {
		t.Fatalf("must delete by the resolved id; log=%v", log)
	}
}

// TestRegisterVMwareDiskTeardownUnresolvedDeletesAllOnVM: an UNRESOLVED disk
// (id never came back; the VMware create carries no name to find it by) is
// recovered by deleting EVERY disk on the (run-unique, ours, created-diskless) VM.
// Mutation: drop the find/delete-all loop → the created-but-unresolved disk
// orphans.
func TestRegisterVMwareDiskTeardownUnresolvedDeletesAllOnVM(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareDiskTeardown(cl, fakeVMwareDiskSeam{log: &log, ids: []string{"disk-1", "disk-2"}},
		&vmwareDiskTeardownRef{VMID: "vm-1"}) // unresolved
	cl.TeardownAll(context.Background())
	if indexOf(log, "disk.find:vm-1") == -1 || indexOf(log, "disk.delete:disk-1") == -1 || indexOf(log, "disk.delete:disk-2") == -1 {
		t.Fatalf("unresolved disk must list the VM and delete every disk; log=%v", log)
	}
}

// TestVMwareLeafTeardownDeferredNoopWhenVMGone pins the deferred-after-explicit
// case: when the explicit deprovision already swept a leaf and deleted the VM, the
// deferred (unresolved) disk/adapter teardown relists the now-gone VM. With the
// realistic gone-VM listing (empty), the teardown is a clean NO-OP — no false
// teardown failure, nothing deleted. (A non-200 listing on a gone VM would instead
// surface as a failure: fail-safe, never an orphan, since the explicit phase
// already removed the leaf while the VM existed.)
func TestVMwareLeafTeardownDeferredNoopWhenVMGone(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareDiskTeardown(cl, fakeVMwareDiskSeam{log: &log, ids: nil}, // VM gone → empty listing
		&vmwareDiskTeardownRef{VMID: "vm-gone"})
	registerVMwareNetworkAdapterTeardown(cl, fakeVMwareAdapterSeam{log: &log, ids: nil},
		&vmwareAdapterTeardownRef{VMID: "vm-gone"})
	if f := cl.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("deferred leaf teardown must be a clean no-op when the VM is gone, got failures %v", f)
	}
	if indexOf(log, "disk.delete") != -1 || indexOf(log, "adapter.delete") != -1 {
		t.Fatalf("nothing to delete when the VM-scoped listing is empty; log=%v", log)
	}
}

// TestRegisterVMwareNetworkAdapterTeardownResolvedDeletesByID: a resolved adapter
// is deleted by id, no VM-wide listing.
func TestRegisterVMwareNetworkAdapterTeardownResolvedDeletesByID(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareNetworkAdapterTeardown(cl, fakeVMwareAdapterSeam{log: &log, ids: []string{"should-not-be-used"}},
		&vmwareAdapterTeardownRef{VMID: "vm-1", ID: "ad-3", Resolved: true})
	cl.TeardownAll(context.Background())
	if indexOf(log, "adapter.find") != -1 {
		t.Fatalf("a resolved adapter must delete by id, not list the VM; log=%v", log)
	}
	if indexOf(log, "adapter.delete:ad-3") == -1 {
		t.Fatalf("must delete the resolved adapter id; log=%v", log)
	}
}

// TestRegisterVMwareNetworkAdapterTeardownUnresolvedDeletesAllOnVM: an UNRESOLVED
// adapter (id never came back, MAC platform-assigned) is recovered by deleting
// EVERY adapter on the (run-unique, ours) VM. Mutation: drop the delete-all loop →
// the created-but-unresolved adapter orphans.
func TestRegisterVMwareNetworkAdapterTeardownUnresolvedDeletesAllOnVM(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareNetworkAdapterTeardown(cl, fakeVMwareAdapterSeam{log: &log, ids: []string{"ad-1", "ad-2"}},
		&vmwareAdapterTeardownRef{VMID: "vm-1"}) // unresolved
	cl.TeardownAll(context.Background())
	if indexOf(log, "adapter.find:vm-1") == -1 || indexOf(log, "adapter.delete:ad-1") == -1 || indexOf(log, "adapter.delete:ad-2") == -1 {
		t.Fatalf("unresolved adapter must list the VM and delete every adapter; log=%v", log)
	}
}

// TestVMwareLifecycleTeardownLIFOOrder pins leaves-first: registering VM, then
// disk, then adapter (the cycle's creation order) makes TeardownAll remove them in
// REVERSE — adapter, then disk, then VM (anchor last). Mutation: register the VM
// teardown AFTER the disk/adapter → LIFO deletes the VM before its devices → RED.
func TestVMwareLifecycleTeardownLIFOOrder(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMwareVMTeardown(cl, fakeVMwareVMSeam{log: &log}, &vmwareVMTeardownRef{Name: "vm", DatacenterID: "dc-1", ID: "vm-1", Resolved: true})
	registerVMwareDiskTeardown(cl, fakeVMwareDiskSeam{log: &log}, &vmwareDiskTeardownRef{VMID: "vm-1", ID: "disk-1", Resolved: true})
	registerVMwareNetworkAdapterTeardown(cl, fakeVMwareAdapterSeam{log: &log}, &vmwareAdapterTeardownRef{VMID: "vm-1", ID: "ad-1", Resolved: true})

	cl.TeardownAll(context.Background())

	a, d, v := indexOf(log, "adapter.delete"), indexOf(log, "disk.delete"), indexOf(log, "vm.delete")
	if a == -1 || d == -1 || v == -1 || !(a < d && d < v) {
		t.Fatalf("LIFO must tear down leaves-first (adapter < disk < vm), got order %v", log)
	}
}

// TestVMwareDeprovisionSweepsUnresolvedLeavesBeforeVMDelete pins the ordering
// invariant: when a disk/adapter id never resolved, the explicit deprovision must
// list the VM and delete EVERY leaf BEFORE deleting the VM — because a nameless
// VMware leaf can only be found via the VM-scoped listing, which is impossible
// once the VM is gone (the deferred teardown alone cannot save this case). Mutation:
// reorder vmwareDeprovision to delete the VM first, or skip the unresolved sweep →
// the "leaf before VM" assertion goes RED.
func TestVMwareDeprovisionSweepsUnresolvedLeavesBeforeVMDelete(t *testing.T) {
	var log []string
	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	// adapterID and diskID empty (unresolved) — recovery must go VM-scoped.
	vmwareDeprovision(context.Background(), r, computeVMwareLifecycleCycle{},
		"vm-1", "net-1", "", "",
		fakeVMwareAdapterSeam{log: &log, ids: []string{"ad-1"}},
		fakeVMwareDiskSeam{log: &log, ids: []string{"disk-1"}},
		fakeVMwareVMSeam{log: &log})

	a := indexOf(log, "adapter.delete:ad-1")
	d := indexOf(log, "disk.delete:disk-1")
	v := indexOf(log, "vm.delete:vm-1")
	if a == -1 || d == -1 || v == -1 {
		t.Fatalf("every unresolved leaf and the VM must be deleted; log=%v", log)
	}
	if !(a < v && d < v) {
		t.Fatalf("unresolved leaves MUST be swept BEFORE the VM delete (else they orphan); log=%v", log)
	}
}

// TestVMwareDeprovisionKeepsVMWhenLeafDeleteFails pins the anchor-safety rule: if a
// leaf removal FAILS (e.g. a 409), the VM anchor must NOT be deleted — destroying
// it would strand the nameless leaf (no name, and the VM-scoped listing that finds
// it would be gone). The VM is kept for the deferred TeardownAll to retry. Mutation:
// delete the VM regardless of leaf-delete failures → "vm.delete" appears → RED.
func TestVMwareDeprovisionKeepsVMWhenLeafDeleteFails(t *testing.T) {
	var log []string
	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	vmwareDeprovision(context.Background(), r, computeVMwareLifecycleCycle{},
		"vm-1", "net-1", "ad-1", "disk-1",
		fakeVMwareAdapterSeam{log: &log},
		fakeVMwareDiskSeam{log: &log, delErr: errors.New("409 disk busy")},
		fakeVMwareVMSeam{log: &log})
	if indexOf(log, "disk.delete:disk-1") == -1 {
		t.Fatalf("the disk delete must have been attempted; log=%v", log)
	}
	if indexOf(log, "vm.delete") != -1 {
		t.Fatalf("a failed leaf delete MUST NOT delete the VM anchor (it would strand the nameless leaf); log=%v", log)
	}
}

// TestVMwareDeprovisionResolvedDeletesByIDBeforeVM pins that resolved leaves are
// deleted by id (no VM listing) and still before the VM.
func TestVMwareDeprovisionResolvedDeletesByIDBeforeVM(t *testing.T) {
	var log []string
	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	vmwareDeprovision(context.Background(), r, computeVMwareLifecycleCycle{},
		"vm-1", "net-1", "ad-9", "disk-9",
		fakeVMwareAdapterSeam{log: &log, ids: []string{"should-not-be-listed"}},
		fakeVMwareDiskSeam{log: &log, ids: []string{"should-not-be-listed"}},
		fakeVMwareVMSeam{log: &log})
	if indexOf(log, "adapter.find") != -1 || indexOf(log, "disk.find") != -1 {
		t.Fatalf("resolved leaves must delete by id, not list the VM; log=%v", log)
	}
	a, d, v := indexOf(log, "adapter.delete:ad-9"), indexOf(log, "disk.delete:disk-9"), indexOf(log, "vm.delete:vm-1")
	if a == -1 || d == -1 || v == -1 || !(a < v && d < v) {
		t.Fatalf("resolved leaves must delete by id before the VM; log=%v", log)
	}
}

// TestResolveActivityConcernedItemID pins the FAIL-CLOSED VMware create-id
// resolution: exactly one matching-type concerned item returns its id; a nil
// activity, a no-match, or MORE THAN ONE match all ERROR (never a silent "" the
// caller could treat as OK, never a guessed id from an ambiguous activity).
// Mutations: ignore the Type filter → wrong-type returns "n-1" not error → RED;
// return "" instead of error on no-match → RED; take the first on duplicates → RED.
func TestResolveActivityConcernedItemID(t *testing.T) {
	act := &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
		{ID: "n-1", Type: "network_adapter"},
		{ID: "vm-7", Type: "virtual_machine"},
	}}
	if id, err := resolveActivityConcernedItemID(act, "virtual_machine"); err != nil || id != "vm-7" {
		t.Fatalf("exactly one match must return its id, got id=%q err=%v", id, err)
	}
	if _, err := resolveActivityConcernedItemID(act, "virtual_disk"); err == nil {
		t.Fatal("a no-match must ERROR (created id unresolved), got nil — a silent \"\" would be a false OK")
	}
	if _, err := resolveActivityConcernedItemID(nil, "virtual_machine"); err == nil {
		t.Fatal("a nil activity must ERROR, got nil")
	}
	dup := &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
		{ID: "d-1", Type: "virtual_disk"}, {ID: "d-2", Type: "virtual_disk"},
	}}
	if _, err := resolveActivityConcernedItemID(dup, "virtual_disk"); err == nil {
		t.Fatal("two matching concerned items must fail closed (ambiguous), got nil — guessing could delete the wrong resource")
	}
}

// TestComputeVMwareVMSeamFindIDByNameFailsHardOnDuplicate pins that the unresolved
// find-by-name REFUSES to pick a VM when more than one shares the (run-unique)
// name — it fails closed rather than risk deleting the wrong VM.
func TestComputeVMwareVMSeamFindIDByNameFailsHardOnDuplicate(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"a","name":"dup"},{"id":"b","name":"dup"}]`))
	})
	if _, err := (computeVMwareVMSeam{c}).FindIDByName(context.Background(), "dup", "dc-1"); err == nil {
		t.Fatal("two VMs sharing the name must fail closed (refuse to delete the wrong one), got nil error")
	}
}

// TestComputeVMwareDiskSeamFindIDsByVM pins that the disk teardown recovers a
// created-but-id-unresolved disk by listing the VM's disks (VM-scoped, strict),
// since the VMware create carries no caller-chosen name. Every disk on the
// run-unique, created-diskless VM is one we attached. Mutation: scope the listing
// to the wrong key / drop a returned id → the orphan disk is missed → RED.
func TestComputeVMwareDiskSeamFindIDsByVM(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("virtualMachineId"); got != "vm-1" {
			t.Errorf("disk listing must be scoped to the VM, got virtualMachineId=%q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"disk-1"},{"id":"disk-2"}]`))
	})
	ids, err := (computeVMwareDiskSeam{c}).FindIDsByVM(context.Background(), "vm-1")
	if err != nil || len(ids) != 2 || ids[0] != "disk-1" || ids[1] != "disk-2" {
		t.Fatalf("must return every disk attached to the VM, got ids=%v err=%v", ids, err)
	}
}

// TestRegistryHasComputeVMwareLifecycleGated: the cycle is registered, write-typed,
// and gated out unless -write is set.
func TestRegistryHasComputeVMwareLifecycleGated(t *testing.T) {
	reg := buildRegistry()
	sel, gated, err := reg.Select("compute_vmware_lifecycle", false)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(sel) != 0 {
		t.Fatalf("compute_vmware_lifecycle must be GATED without -write, got selected %v", sel)
	}
	if len(gated) != 1 || gated[0] != "compute_vmware_lifecycle" {
		t.Fatalf("compute_vmware_lifecycle must be the gated write cycle, got %v", gated)
	}
	sel, _, _ = reg.Select("compute_vmware_lifecycle", true)
	if len(sel) != 1 || sel[0].Name() != "compute_vmware_lifecycle" || sel[0].Kind() != KindWrite {
		t.Fatalf("compute_vmware_lifecycle must be selectable (write) with -write, got %v", sel)
	}
}

// TestComputeVMwareLifecycleSkipsWhenVMwareAbsent pins the probe-skip: a 4xx on the
// datacenter list (VMware not available on this tenant) records every write step as
// SKIPPED, attempts ZERO create, and registers ZERO teardown — it never fires a
// create blindly. Mutation: remove the 4xx/empty skip guard → the cycle proceeds to
// list hosts/datastores and beyond → the stub flags the unexpected extra call.
func TestComputeVMwareLifecycleSkipsWhenVMwareAbsent(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("no write/create call expected when VMware is absent, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/vcenters/virtual_datacenters") {
			w.WriteHeader(http.StatusForbidden) // VMware not on this tenant
			return
		}
		t.Errorf("only the datacenter list should be called on a VMware-absent tenant, got GET %s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})

	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	if err := (computeVMwareLifecycleCycle{}).Run(context.Background(), c, r); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if cl := r.Cleanup.Pending(); cl != 0 {
		t.Fatalf("no teardown must be registered when VMware is absent, got %d pending", cl)
	}
	// The failure surface must be EXACTLY the datacenter probe — every other op is a
	// clean skip (no per-step false squeak, no create/delete attempt). So the only
	// non-skipped, non-OK op is the datacenter list itself.
	for _, o := range r.Recorder.Ops() {
		if o.Skipped || o.OK {
			continue
		}
		if o.Endpoint != "compute.vmware.virtual_datacenters.list" {
			t.Fatalf("the only failure when VMware is absent must be the datacenter probe, got a failed %s (%+v)", o.Endpoint, o)
		}
	}
	for _, o := range r.Recorder.Ops() {
		if strings.Contains(o.Endpoint, ".create") || strings.Contains(o.Endpoint, ".delete") || strings.Contains(o.Endpoint, ".connect") {
			if !o.Skipped {
				t.Fatalf("%s must be SKIPPED when VMware is absent, got %+v", o.Endpoint, o)
			}
		}
	}
}

// TestComputeVMwareLifecycleSkipsWithoutUsableSubstrate pins that a present
// datacenter with NO host (so no guest-OS moref can be discovered) skips every
// write step and fires ZERO create — it refuses to create a VM without the
// substrate it needs. Mutation: drop the `hostClusterID/datastoreID/osMoref` guard → a
// create POST fires → the stub flags it.
func TestComputeVMwareLifecycleSkipsWithoutUsableSubstrate(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("no create expected without a usable host, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		switch {
		case strings.HasSuffix(r.URL.Path, "/vcenters/virtual_datacenters"):
			_, _ = w.Write([]byte(`[{"id":"dc-1","name":"DC"}]`)) // a datacenter exists
		default:
			_, _ = w.Write([]byte(`[]`)) // but no host / datastore / network / guest-OS
		}
	})

	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	if err := (computeVMwareLifecycleCycle{}).Run(context.Background(), c, r); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if cl := r.Cleanup.Pending(); cl != 0 {
		t.Fatalf("no teardown must be registered without a usable substrate, got %d pending", cl)
	}
	for _, o := range r.Recorder.Ops() {
		if strings.Contains(o.Endpoint, ".create") || strings.Contains(o.Endpoint, ".delete") || strings.Contains(o.Endpoint, ".connect") {
			if !o.Skipped {
				t.Fatalf("%s must be SKIPPED without a usable substrate, got %+v", o.Endpoint, o)
			}
		}
	}
}

// TestComputeVMwareLifecycleIdentityFailureIsObservable pins that a run-identity
// generation failure creates NOTHING and is recorded as a FAILURE op (so the run
// exits non-zero) — not a silent exit-0.
func TestComputeVMwareLifecycleIdentityFailureIsObservable(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no call expected when identity generation fails, got %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	cyc := computeVMwareLifecycleCycle{tokenFunc: func() (string, error) { return "", errors.New("no entropy") }}
	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	_ = cyc.Run(context.Background(), c, r)

	if r.Cleanup.Pending() != 0 {
		t.Fatalf("nothing must be created/registered on an identity failure, got %d pending", r.Cleanup.Pending())
	}
	var failed bool
	for _, o := range r.Recorder.Ops() {
		if o.Endpoint == "compute.vmware.run_identity" && !o.OK && !o.Skipped {
			failed = true
		}
	}
	if !failed {
		t.Fatal("identity failure must be recorded as a failed op (observable, exit non-zero)")
	}
}
