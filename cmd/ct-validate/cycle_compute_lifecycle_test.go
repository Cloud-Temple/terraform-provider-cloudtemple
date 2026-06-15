package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// --- recording fake seams (offline, no client) ---

type fakeVMSeam struct {
	log     *[]string
	findID  string
	findErr error
	delErr  error // when non-nil, DeleteAndWait fails (e.g. an OpenIaaS 403-on-absent)
}

func (f fakeVMSeam) DeleteAndWait(_ context.Context, id string) error {
	*f.log = append(*f.log, "vm.delete:"+id)
	return f.delErr
}
func (f fakeVMSeam) PowerOffAndWait(_ context.Context, id string) error {
	*f.log = append(*f.log, "vm.poweroff:"+id)
	return nil
}
func (f fakeVMSeam) FindIDByName(_ context.Context, name, mmID string) (string, error) {
	*f.log = append(*f.log, "vm.find:"+name+"@"+mmID)
	return f.findID, f.findErr
}

type fakeDiskSeam struct {
	log         *[]string
	connections []string
	findID      string
	delErr      error // when non-nil, DeleteAndWait fails (e.g. an OpenIaaS 403-on-absent)
}

func (f fakeDiskSeam) ReadConnections(_ context.Context, id string) ([]string, error) {
	return f.connections, nil
}
func (f fakeDiskSeam) DisconnectAndWait(_ context.Context, id, vmID string) error {
	*f.log = append(*f.log, "disk.disconnect:"+id+"/"+vmID)
	return nil
}
func (f fakeDiskSeam) DeleteAndWait(_ context.Context, id string) error {
	*f.log = append(*f.log, "disk.delete:"+id)
	return f.delErr
}
func (f fakeDiskSeam) FindIDByName(_ context.Context, name, vmID string) (string, error) {
	*f.log = append(*f.log, "disk.find:"+name)
	return f.findID, nil
}

type fakeAdapterSeam struct {
	log    *[]string
	ids    []string // FindIDsByVM returns these
	delErr error    // when non-nil, DeleteAndWait fails (e.g. an OpenIaaS 403-on-absent)
}

func (f fakeAdapterSeam) FindIDsByVM(_ context.Context, vmID string) ([]string, error) {
	*f.log = append(*f.log, "adapter.find:"+vmID)
	return f.ids, nil
}
func (f fakeAdapterSeam) DeleteAndWait(_ context.Context, id string) error {
	*f.log = append(*f.log, "adapter.delete:"+id)
	return f.delErr
}

func indexOf(log []string, prefix string) int {
	for i, s := range log {
		if strings.HasPrefix(s, prefix) {
			return i
		}
	}
	return -1
}

// TestRegisterVMTeardownResolvedDeletesByID: a resolved ref deletes by id
// (power-off best-effort first), never falling back to find-by-name. Mutation:
// drop the `ref.Resolved` check → find-by-name is used → "vm.find" appears.
func TestRegisterVMTeardownResolvedDeletesByID(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMTeardown(cl, fakeVMSeam{log: &log, findID: "should-not-be-used"},
		&vmTeardownRef{Name: "vm-a", MachineManagerID: "mm-1", ID: "vm-id-1", Resolved: true})
	cl.TeardownAll(context.Background())
	if indexOf(log, "vm.find") != -1 {
		t.Fatalf("a resolved ref must delete by id, not find by name; log=%v", log)
	}
	if indexOf(log, "vm.delete:vm-id-1") == -1 {
		t.Fatalf("must delete by the resolved id; log=%v", log)
	}
}

// TestRegisterVMTeardownUnresolvedFindsByName: an UNRESOLVED ref (create id
// never came back) finds the VM by its deterministic name and deletes it — this
// is why the teardown is registered BEFORE the create. Mutation: register after
// the create (so the ref is never set) is exactly this path; dropping find →
// orphan survives.
func TestRegisterVMTeardownUnresolvedFindsByName(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMTeardown(cl, fakeVMSeam{log: &log, findID: "found-vm-9"},
		&vmTeardownRef{Name: "vm-b", MachineManagerID: "mm-1"}) // not resolved
	cl.TeardownAll(context.Background())
	if indexOf(log, "vm.find:vm-b@mm-1") == -1 || indexOf(log, "vm.delete:found-vm-9") == -1 {
		t.Fatalf("unresolved ref must find-by-name then delete; log=%v", log)
	}
}

// TestRegisterVMTeardownNotFoundIsNoop: unresolved + not found → nothing to do,
// no delete (idempotent: never created / already gone).
func TestRegisterVMTeardownNotFoundIsNoop(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMTeardown(cl, fakeVMSeam{log: &log, findID: ""},
		&vmTeardownRef{Name: "vm-c", MachineManagerID: "mm-1"})
	if f := cl.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("not-found teardown must be a no-op success, got failures %v", f)
	}
	if indexOf(log, "vm.delete") != -1 {
		t.Fatalf("nothing must be deleted when not found; log=%v", log)
	}
}

// TestRegisterVirtualDiskTeardownDisconnectsEveryConnectionThenDeletes pins the
// disk doctrine: a user disk is disconnected from EVERY connected VM BEFORE
// delete (a VM delete never cascades it). Mutation: drop the disconnect loop →
// the "disconnect before delete" assertion goes RED.
func TestRegisterVirtualDiskTeardownDisconnectsEveryConnectionThenDeletes(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVirtualDiskTeardown(cl, fakeDiskSeam{log: &log, connections: []string{"vm-1", "vm-2"}},
		&diskTeardownRef{Name: "d-data", VMID: "vm-1", ID: "disk-7", Resolved: true})
	cl.TeardownAll(context.Background())
	d1 := indexOf(log, "disk.disconnect:disk-7/vm-1")
	d2 := indexOf(log, "disk.disconnect:disk-7/vm-2")
	del := indexOf(log, "disk.delete:disk-7")
	if d1 == -1 || d2 == -1 || del == -1 || d1 > del || d2 > del {
		t.Fatalf("disk must disconnect EVERY connection before delete; log=%v", log)
	}
}

// TestRegisterNetworkAdapterTeardownResolvedDeletesByID: a resolved adapter is
// deleted by id, no VM-wide listing.
func TestRegisterNetworkAdapterTeardownResolvedDeletesByID(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerNetworkAdapterTeardown(cl, fakeAdapterSeam{log: &log, ids: []string{"should-not-be-used"}},
		&adapterTeardownRef{VMID: "vm-1", ID: "ad-3", Resolved: true})
	cl.TeardownAll(context.Background())
	if indexOf(log, "adapter.find") != -1 {
		t.Fatalf("a resolved adapter must delete by id, not list the VM; log=%v", log)
	}
	if indexOf(log, "adapter.delete:ad-3") == -1 {
		t.Fatalf("must delete the resolved adapter id; log=%v", log)
	}
}

// TestRegisterNetworkAdapterTeardownUnresolvedDeletesAllOnVM: an UNRESOLVED
// adapter (create id never came back, and no MAC since the platform assigns it)
// is recovered by deleting EVERY adapter on the (run-unique, ours) VM. Mutation:
// drop the find/delete-all loop → the created-but-unresolved adapter orphans.
func TestRegisterNetworkAdapterTeardownUnresolvedDeletesAllOnVM(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerNetworkAdapterTeardown(cl, fakeAdapterSeam{log: &log, ids: []string{"ad-1", "ad-2"}},
		&adapterTeardownRef{VMID: "vm-1"}) // unresolved
	cl.TeardownAll(context.Background())
	if indexOf(log, "adapter.find:vm-1") == -1 || indexOf(log, "adapter.delete:ad-1") == -1 || indexOf(log, "adapter.delete:ad-2") == -1 {
		t.Fatalf("unresolved adapter must list the VM and delete every adapter; log=%v", log)
	}
}

// TestComputeLifecycleTeardownLIFOOrder pins leaves-first: registering VM, then
// disk, then adapter (the cycle's creation order) makes TeardownAll remove them
// in REVERSE — adapter, then disk, then VM (anchor last). Mutation: register the
// VM teardown AFTER the disk/adapter → LIFO would delete the VM before its
// devices → this order assertion goes RED.
func TestComputeLifecycleTeardownLIFOOrder(t *testing.T) {
	var log []string
	cl := NewCleanup()
	registerVMTeardown(cl, fakeVMSeam{log: &log}, &vmTeardownRef{Name: "vm", ID: "vm-1", Resolved: true})
	registerVirtualDiskTeardown(cl, fakeDiskSeam{log: &log}, &diskTeardownRef{Name: "d", VMID: "vm-1", ID: "disk-1", Resolved: true})
	registerNetworkAdapterTeardown(cl, fakeAdapterSeam{log: &log}, &adapterTeardownRef{VMID: "vm-1", ID: "ad-1", Resolved: true})

	cl.TeardownAll(context.Background())

	a, d, v := indexOf(log, "adapter.delete"), indexOf(log, "disk.delete"), indexOf(log, "vm.delete")
	if a == -1 || d == -1 || v == -1 || !(a < d && d < v) {
		t.Fatalf("LIFO must tear down leaves-first (adapter < disk < vm), got order %v", log)
	}
}

// TestVMSizeUsesTemplateWhenLarger pins the order-constraint fix: the create size
// is max(floor, template). The template's own CPU/RAM must win when it exceeds the
// floor (deploying a template under-sized is rejected: MEMORY_CONSTRAINT_VIOLATION_ORDER);
// the floor guards a 0/absent template value. Mutation: always return the floor →
// the "template larger" case goes RED (and the live create would 400/violate).
func TestVMSizeUsesTemplateWhenLarger(t *testing.T) {
	const floor = 1073741824 // 1 GiB
	if got := vmSize(floor, 4*floor); got != 4*floor {
		t.Fatalf("template larger than floor must win (avoids the order constraint), got %d", got)
	}
	if got := vmSize(floor, 0); got != floor {
		t.Fatalf("a 0/absent template value must fall back to the floor, got %d", got)
	}
	if got := vmSize(floor, floor/2); got != floor {
		t.Fatalf("a template smaller than the floor must keep the floor, got %d", got)
	}
	if got := vmSize(2, 8); got != 8 { // CPU dimension
		t.Fatalf("cpu sizing must also take the larger, got %d", got)
	}
}

// TestOpenIaaSVMCreateReqUsesSelectedTemplateSizing pins the integration the unit
// test alone misses: the SELECTED template's id AND its CPU/Memory (sized to the
// floor) actually reach the create request — a regression that hardcoded the floor
// constants or read a different template would be caught here. Mutation: build the
// request with clVMCPU/clVMMemory instead of the template values → RED.
func TestOpenIaaSVMCreateReqUsesSelectedTemplateSizing(t *testing.T) {
	const bigMem = 4 * clVMMemory // template wants 4 GiB, above the 1 GiB floor
	req := openIaaSVMCreateReq("ct-validate-vm", "tmpl-selected", 4, bigMem)
	if req.TemplateID != "tmpl-selected" {
		t.Fatalf("must deploy from the selected template, got %q", req.TemplateID)
	}
	if req.CPU != 4 || req.Memory != bigMem {
		t.Fatalf("must size from the template (cpu=4, mem=%d), got cpu=%d mem=%d", bigMem, req.CPU, req.Memory)
	}
	// A template reporting 0 (listing did not populate it) falls back to the floor.
	floorReq := openIaaSVMCreateReq("ct-validate-vm", "tmpl-zero", 0, 0)
	if floorReq.CPU != clVMCPU || floorReq.Memory != clVMMemory {
		t.Fatalf("a 0 template must fall back to the floor, got cpu=%d mem=%d", floorReq.CPU, floorReq.Memory)
	}
}

// TestSelectOpenIaaSDiskSR pins the disk-SR selection that fixes the live
// "could not find storage repository" failure: a SHARED SR in the VM's pool is
// preferred; a non-shared SR is only chosen when it is LOCAL to the VM's host;
// SRs in another pool, in maintenance, inaccessible, or too small are rejected;
// none-fits yields "". Mutation: ignore pool/host (return the first usable) → the
// wrong-pool SR is chosen → RED.
func TestSelectOpenIaaSDiskSR(t *testing.T) {
	const need = 1073741824
	bo := func(id string) client.BaseObject { return client.BaseObject{ID: id} }
	srs := []*client.OpenIaaSStorageRepository{
		{ID: "sr-othpool", Shared: true, Accessible: 1, FreeCapacity: 10 * need, Pool: bo("pool-OTHER")}, // shared but wrong pool
		{ID: "sr-maint", Shared: true, MaintenanceMode: true, Accessible: 1, FreeCapacity: 10 * need, Pool: bo("pool-A")},
		{ID: "sr-small", Shared: true, Accessible: 1, FreeCapacity: need / 2, Pool: bo("pool-A")},   // too small
		{ID: "sr-shared", Shared: true, Accessible: 1, FreeCapacity: 10 * need, Pool: bo("pool-A")}, // the right one
		{ID: "sr-local", Shared: false, Accessible: 1, FreeCapacity: 10 * need, Host: bo("host-1")},
	}
	if got := selectOpenIaaSDiskSR(srs, "pool-A", "host-1", need); got != "sr-shared" {
		t.Fatalf("must pick the SHARED SR in the VM's pool, got %q", got)
	}
	// No shared SR in the pool → fall back to a local SR on the VM's host.
	noShared := []*client.OpenIaaSStorageRepository{
		{ID: "sr-othpool", Shared: true, Accessible: 1, FreeCapacity: 10 * need, Pool: bo("pool-OTHER")},
		{ID: "sr-local", Shared: false, Accessible: 1, FreeCapacity: 10 * need, Host: bo("host-1")},
	}
	if got := selectOpenIaaSDiskSR(noShared, "pool-A", "host-1", need); got != "sr-local" {
		t.Fatalf("must fall back to a host-local SR, got %q", got)
	}
	// Nothing in the VM's pool/host → "" (the cycle then surfaces it, not silent).
	if got := selectOpenIaaSDiskSR(srs, "pool-NONE", "host-NONE", need); got != "" {
		t.Fatalf("no reachable SR must yield \"\", got %q", got)
	}
	// VM read yielded no pool/host (empty) → "" even with otherwise-valid SRs (we
	// must NOT fall back to an arbitrary SR — that is the original bug).
	if got := selectOpenIaaSDiskSR(srs, "", "", need); got != "" {
		t.Fatalf("an unknown VM pool/host must yield \"\" (no blind pick), got %q", got)
	}
	// An inaccessible shared SR in the pool is skipped in favour of the next valid one.
	inacc := []*client.OpenIaaSStorageRepository{
		{ID: "sr-inacc", Shared: true, Accessible: 0, FreeCapacity: 10 * need, Pool: bo("pool-A")}, // accessible=0 → rejected
		{ID: "sr-ok", Shared: true, Accessible: 1, FreeCapacity: 10 * need, Pool: bo("pool-A")},
	}
	if got := selectOpenIaaSDiskSR(inacc, "pool-A", "host-1", need); got != "sr-ok" {
		t.Fatalf("an inaccessible SR must be skipped for the next valid one, got %q", got)
	}
	// Host-local fallback must match the VM's OWN host, not another host's local SR.
	hosts := []*client.OpenIaaSStorageRepository{
		{ID: "sr-otherhost", Shared: false, Accessible: 1, FreeCapacity: 10 * need, Host: bo("host-2")},
		{ID: "sr-myhost", Shared: false, Accessible: 1, FreeCapacity: 10 * need, Host: bo("host-1")},
	}
	if got := selectOpenIaaSDiskSR(hosts, "pool-A", "host-1", need); got != "sr-myhost" {
		t.Fatalf("host-local fallback must match the VM's own host, got %q", got)
	}
	// A host-local SR appearing BEFORE a shared-in-pool SR must NOT win: shared-in-pool
	// is always preferred regardless of list order.
	order := []*client.OpenIaaSStorageRepository{
		{ID: "sr-local-first", Shared: false, Accessible: 1, FreeCapacity: 10 * need, Host: bo("host-1")},
		{ID: "sr-shared-later", Shared: true, Accessible: 1, FreeCapacity: 10 * need, Pool: bo("pool-A")},
	}
	if got := selectOpenIaaSDiskSR(order, "pool-A", "host-1", need); got != "sr-shared-later" {
		t.Fatalf("shared-in-pool must win over an earlier host-local SR, got %q", got)
	}
}

// TestRegistryHasComputeLifecycleGated: the cycle is registered, write-typed, and
// gated out unless -write is set.
func TestRegistryHasComputeLifecycleGated(t *testing.T) {
	reg := buildRegistry()
	sel, gated, err := reg.Select("compute_lifecycle", false)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(sel) != 0 {
		t.Fatalf("compute_lifecycle must be GATED without -write, got selected %v", sel)
	}
	if len(gated) != 1 || gated[0] != "compute_lifecycle" {
		t.Fatalf("compute_lifecycle must be the gated write cycle, got %v", gated)
	}
	sel, _, _ = reg.Select("compute_lifecycle", true)
	if len(sel) != 1 || sel[0].Name() != "compute_lifecycle" || sel[0].Kind() != KindWrite {
		t.Fatalf("compute_lifecycle must be selectable (write) with -write, got %v", sel)
	}
}

// TestComputeLifecycleSkipsWithoutSubstrate pins that with NO machine manager,
// the cycle records every write step as SKIPPED and attempts ZERO create — it
// never guesses or fires a create blindly. Mutation: turn a skip into a create
// attempt → the stub flags the unexpected POST.
func TestComputeLifecycleSkipsWithoutSubstrate(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("no write/create call expected without a machine manager, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if isOpenIaaSMachineManagers(r.URL.Path) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`)) // no machine manager
			return
		}
		t.Errorf("only the machine-managers list should be called, got GET %s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})

	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	if err := (computeLifecycleCycle{}).Run(context.Background(), c, r); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if cl := r.Cleanup.Pending(); cl != 0 {
		t.Fatalf("no teardown must be registered when there is no substrate, got %d pending", cl)
	}
	for _, o := range r.Recorder.Ops() {
		if strings.Contains(o.Endpoint, ".create") || strings.Contains(o.Endpoint, ".delete") || strings.Contains(o.Endpoint, ".connect") {
			if !o.Skipped {
				t.Fatalf("%s must be SKIPPED without substrate, got %+v", o.Endpoint, o)
			}
		}
	}
}

// TestComputeVMSeamFindIDByNameFailsHardOnDuplicate pins that the unresolved
// find-by-name REFUSES to pick a VM when more than one shares the (run-unique)
// name — it fails closed rather than risk deleting the wrong VM. Mutation: have
// FindIDByName return the first match → no error → RED.
func TestComputeVMSeamFindIDByNameFailsHardOnDuplicate(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"a","name":"dup"},{"id":"b","name":"dup"}]`))
	})
	if _, err := (computeVMSeam{c}).FindIDByName(context.Background(), "dup", "mm-1"); err == nil {
		t.Fatal("two VMs sharing the name must fail closed (refuse to delete the wrong one), got nil error")
	}
}

// TestComputeVirtualDiskSeamFindTenantWideWhenUnattached pins the anti-orphan
// fallback: a created-but-UNATTACHED disk (absent from the VM-scoped listing) is
// still found tenant-wide by its run-unique name. Mutation: drop the tenant-wide
// pass → the unattached disk is not found → would orphan → RED.
func TestComputeVirtualDiskSeamFindTenantWideWhenUnattached(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.URL.Query().Get("virtualMachineId") != "" {
			_, _ = w.Write([]byte(`[]`)) // VM-scoped: not attached → empty
			return
		}
		_, _ = w.Write([]byte(`[{"id":"disk-9","name":"d-data"}]`)) // tenant-wide: found
	})
	id, err := (computeVirtualDiskSeam{c}).FindIDByName(context.Background(), "d-data", "vm-1")
	if err != nil || id != "disk-9" {
		t.Fatalf("a created-but-unattached disk must be found tenant-wide, got id=%q err=%v", id, err)
	}
}

// TestComputeVirtualDiskSeamFindFailsHardOnDuplicate pins fail-closed on a
// duplicate disk name (refuse to delete the wrong disk).
func TestComputeVirtualDiskSeamFindFailsHardOnDuplicate(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.URL.Query().Get("virtualMachineId") != "" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`[{"id":"a","name":"d-data"},{"id":"b","name":"d-data"}]`))
	})
	if _, err := (computeVirtualDiskSeam{c}).FindIDByName(context.Background(), "d-data", "vm-1"); err == nil {
		t.Fatal("two disks sharing the name must fail closed, got nil error")
	}
}

// TestComputeLifecycleIdentityFailureIsObservable pins that a run-identity
// generation failure (no entropy) creates NOTHING and is recorded as a FAILURE
// op (so the run exits non-zero) — not a silent exit-0. Mutation: revert the
// token step to a bare `return err` (engine-swallowed) → no failure op recorded
// → RED.
func TestComputeLifecycleIdentityFailureIsObservable(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no call expected when identity generation fails, got %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	})
	cyc := computeLifecycleCycle{tokenFunc: func() (string, error) { return "", errors.New("no entropy") }}
	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	_ = cyc.Run(context.Background(), c, r)

	if r.Cleanup.Pending() != 0 {
		t.Fatalf("nothing must be created/registered on an identity failure, got %d pending", r.Cleanup.Pending())
	}
	var failed bool
	for _, o := range r.Recorder.Ops() {
		if o.Endpoint == "compute.openiaas.run_identity" && !o.OK && !o.Skipped {
			failed = true
		}
	}
	if !failed {
		t.Fatal("identity failure must be recorded as a failed op (observable, exit non-zero)")
	}
}

// TestFlagsValidateRejectsRunawayLoad pins the hard ceilings on the load knobs: a
// typo'd -concurrency / -runs is refused before anything runs. Mutation: drop the
// upper-bound checks in validate() → these are accepted → RED.
func TestFlagsValidateRejectsRunawayLoad(t *testing.T) {
	out, err := os.CreateTemp(t.TempDir(), "usage")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	defer out.Close()
	if _, err := parseFlags([]string{"-concurrency", "5000"}, out); err == nil {
		t.Fatal("-concurrency 5000 must be rejected (runaway load)")
	}
	if _, err := parseFlags([]string{"-runs", "100000000"}, out); err == nil {
		t.Fatal("-runs 100000000 must be rejected (runaway load)")
	}
}

// --- Bug B: OpenIaaS deferred teardown 403-on-absent (#303) via same-cycle proof ---

// TestConfirmComputeDeleteByPriorDelete pins the pure decision: nil/404 are
// idempotent success; a 403 is accepted ONLY with same-cycle explicit-delete proof,
// else it FAILS CLOSED; any other error surfaces. Mutations: accept a 403 without
// proof → the "no proof" case wrongly passes → RED; reject a proven 403 → the
// proven case wrongly fails → RED.
func TestConfirmComputeDeleteByPriorDelete(t *testing.T) {
	forbidden := client.StatusError{Code: http.StatusForbidden, Body: "Forbidden."}
	notFound := client.StatusError{Code: http.StatusNotFound}
	serverErr := client.StatusError{Code: http.StatusInternalServerError}

	if confirmComputeDeleteByPriorDelete(nil, false, "id-x") != nil {
		t.Fatal("a nil delete error must be success")
	}
	if confirmComputeDeleteByPriorDelete(notFound, false, "id-x") != nil {
		t.Fatal("a definitive 404 must be idempotent success even without prior-delete proof")
	}
	if err := confirmComputeDeleteByPriorDelete(forbidden, true, "id-x"); err != nil {
		t.Fatalf("a 403 WITH same-cycle explicit-delete proof must be success (no false orphan), got %v", err)
	}
	if confirmComputeDeleteByPriorDelete(forbidden, false, "id-x") == nil {
		t.Fatal("a 403 WITHOUT proof must FAIL CLOSED (a possible orphan, never masked)")
	}
	if confirmComputeDeleteByPriorDelete(serverErr, true, "id-x") == nil {
		t.Fatal("a non-403/404 error must surface even with prior-delete proof")
	}
}

// TestOpenIaaSVMTeardownConfirms403ByPriorDelete pins the wiring: the deferred VM
// teardown re-deletes an already-gone VM (OpenIaaS answers 403, not 404). With the
// same-cycle explicit-delete proof it is a clean success (no false orphan); without
// it, it fails closed. Mutation: revert the wrap to idempotentDeleteErr (404-only)
// → the proven 403 surfaces as a failure → RED.
func TestOpenIaaSVMTeardownConfirms403ByPriorDelete(t *testing.T) {
	forbidden := client.StatusError{Code: http.StatusForbidden, Body: "Forbidden."}

	// VM explicitly deleted this cycle: deferred delete 403s, proof present → success.
	var log []string
	cl := NewCleanup()
	registerVMTeardown(cl, fakeVMSeam{log: &log, delErr: forbidden},
		&vmTeardownRef{Name: "vm", ID: "vm-1", Resolved: true, ExplicitlyDeleted: true})
	if f := cl.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("a 403 on a VM proven deleted this cycle must be confirmed absent (no false orphan), got failures %v", f)
	}

	// No proof (e.g. explicit delete skipped by the breaker): 403 fails closed.
	var log2 []string
	cl2 := NewCleanup()
	registerVMTeardown(cl2, fakeVMSeam{log: &log2, delErr: forbidden},
		&vmTeardownRef{Name: "vm", ID: "vm-1", Resolved: true, ExplicitlyDeleted: false})
	if f := cl2.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("a 403 with NO same-cycle delete proof must fail closed (possible orphan), got failures %v", f)
	}

	// A definitive 404 is idempotent success regardless of proof.
	var log3 []string
	cl3 := NewCleanup()
	registerVMTeardown(cl3, fakeVMSeam{log: &log3, delErr: client.StatusError{Code: http.StatusNotFound}},
		&vmTeardownRef{Name: "vm", ID: "vm-1", Resolved: true, ExplicitlyDeleted: false})
	if f := cl3.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("a definitive 404 must be idempotent success, got failures %v", f)
	}

	// UNRESOLVED (create id never came back): the teardown finds the VM by name and
	// re-deletes it; a 403 there carries NO same-cycle proof → must fail closed (a
	// created-but-unresolved, still-present VM must never be masked). Mutation: force
	// priorDeleteOK=true on this path → the 403 is masked → RED.
	var log4 []string
	cl4 := NewCleanup()
	registerVMTeardown(cl4, fakeVMSeam{log: &log4, findID: "found-vm-9", delErr: forbidden},
		&vmTeardownRef{Name: "vm", MachineManagerID: "mm-1"}) // Resolved=false
	if f := cl4.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("an unresolved find-by-name VM 403 carries no proof and must fail closed, got failures %v", f)
	}
}

// TestOpenIaaSDiskTeardownConfirms403ByPriorDelete: same doctrine for the disk leaf.
func TestOpenIaaSDiskTeardownConfirms403ByPriorDelete(t *testing.T) {
	forbidden := client.StatusError{Code: http.StatusForbidden, Body: "Forbidden."}

	var log []string
	cl := NewCleanup()
	registerVirtualDiskTeardown(cl, fakeDiskSeam{log: &log, delErr: forbidden},
		&diskTeardownRef{Name: "d-data", VMID: "vm-1", ID: "disk-1", Resolved: true, ExplicitlyDeleted: true})
	if f := cl.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("a 403 on a disk proven deleted this cycle must be confirmed absent, got failures %v", f)
	}

	var log2 []string
	cl2 := NewCleanup()
	registerVirtualDiskTeardown(cl2, fakeDiskSeam{log: &log2, delErr: forbidden},
		&diskTeardownRef{Name: "d-data", VMID: "vm-1", ID: "disk-1", Resolved: true, ExplicitlyDeleted: false})
	if f := cl2.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("a disk 403 with NO same-cycle delete proof must fail closed, got failures %v", f)
	}

	// UNRESOLVED: the disk is found by name (created-but-unresolved), then re-deleted;
	// a 403 there carries no proof → must fail closed. Mutation: force priorDeleteOK=true
	// on the find-by-name path → the 403 is masked → RED.
	var log3 []string
	cl3 := NewCleanup()
	registerVirtualDiskTeardown(cl3, fakeDiskSeam{log: &log3, findID: "disk-9", delErr: forbidden},
		&diskTeardownRef{Name: "d-data", VMID: "vm-1"}) // Resolved=false
	if f := cl3.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("an unresolved find-by-name disk 403 carries no proof and must fail closed, got failures %v", f)
	}
}

// TestOpenIaaSAdapterTeardownConfirms403ByPriorDelete: the adapter leaf, both the
// resolved (proof-bearing) path and the unresolved delete-all-on-VM path (which
// carries NO per-id proof → a 403 there must fail closed).
func TestOpenIaaSAdapterTeardownConfirms403ByPriorDelete(t *testing.T) {
	forbidden := client.StatusError{Code: http.StatusForbidden, Body: "Forbidden."}

	// Resolved + proof → 403 confirmed absent.
	var log []string
	cl := NewCleanup()
	registerNetworkAdapterTeardown(cl, fakeAdapterSeam{log: &log, delErr: forbidden},
		&adapterTeardownRef{VMID: "vm-1", ID: "ad-1", Resolved: true, ExplicitlyDeleted: true})
	if f := cl.TeardownAll(context.Background()); len(f) != 0 {
		t.Fatalf("a 403 on an adapter proven deleted this cycle must be confirmed absent, got failures %v", f)
	}

	// Resolved + no proof → fail closed.
	var log2 []string
	cl2 := NewCleanup()
	registerNetworkAdapterTeardown(cl2, fakeAdapterSeam{log: &log2, delErr: forbidden},
		&adapterTeardownRef{VMID: "vm-1", ID: "ad-1", Resolved: true, ExplicitlyDeleted: false})
	if f := cl2.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("an adapter 403 with NO same-cycle delete proof must fail closed, got failures %v", f)
	}

	// Unresolved delete-all-on-VM: no per-id proof → a 403 must fail closed even if
	// the ref were (wrongly) marked. Mutation: pass ref.ExplicitlyDeleted instead of
	// false in the delete-all branch → this 403 would be masked → RED.
	var log3 []string
	cl3 := NewCleanup()
	registerNetworkAdapterTeardown(cl3, fakeAdapterSeam{log: &log3, ids: []string{"ad-9"}, delErr: forbidden},
		&adapterTeardownRef{VMID: "vm-1", ExplicitlyDeleted: true}) // unresolved (Resolved=false)
	if f := cl3.TeardownAll(context.Background()); len(f) != 1 {
		t.Fatalf("an unresolved delete-all 403 must fail closed (no per-id proof), got failures %v", f)
	}
}

// TestExplicitDeleteMarksProofOnlyWhenClosureRuns is the blocker-#2 guard: the
// same-cycle proof must be set from INSIDE the breaker-gated closure, never from
// r.op's return value (which is nil on a breaker SKIP too). A skipped explicit
// delete must leave the proof false, so a later 403-on-absent fails closed instead
// of masking a still-present forbidden orphan. Mutation: set the flag with
// `r.op(...) == nil` → case (a) would mark it true on a skip → RED.
func TestExplicitDeleteMarksProofOnlyWhenClosureRuns(t *testing.T) {
	cyc := computeLifecycleCycle{}

	// (a) breaker tripped → op skipped → closure NOT run → proof stays false.
	b := NewBreaker(1000, 0.99, 1000)
	b.Trip("force-skip")
	r := &Run{Recorder: NewRecorder(), Breaker: b, Cleanup: NewCleanup()}
	deleted, ran := false, false
	cyc.explicitDelete(r, &deleted, "compute.openiaas.virtual_machine.delete", func() error {
		ran = true
		return nil
	})
	if ran {
		t.Fatal("a tripped breaker must skip the op — the closure must not run")
	}
	if deleted {
		t.Fatal("a skipped (never-run) explicit delete must NOT set the same-cycle proof (else a later 403 masks an orphan)")
	}

	// (b) breaker allows + delete succeeds → proof set.
	r2 := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	deleted2 := false
	cyc.explicitDelete(r2, &deleted2, "compute.openiaas.virtual_machine.delete", func() error { return nil })
	if !deleted2 {
		t.Fatal("a ran-and-succeeded explicit delete must set the same-cycle proof")
	}

	// (c) breaker allows + delete fails → proof NOT set.
	r3 := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	deleted3 := false
	cyc.explicitDelete(r3, &deleted3, "compute.openiaas.virtual_machine.delete", func() error {
		return errors.New("delete boom")
	})
	if deleted3 {
		t.Fatal("a failed explicit delete must NOT set the same-cycle proof")
	}
}
