package main

import (
	"context"
	"net/http"
	"testing"
)

// TestMachineManagersCycleProbesIdentityThenList pins that the cycle performs
// EXACTLY two recorded ops, in order — the local run-identity then the OpenIaaS
// machine_managers list — and nothing else. Mutation: drop either op → the count
// or order assertion goes RED.
func TestMachineManagersCycleProbesIdentityThenList(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// machine_managers list = GET /api/compute/v1/open_iaas
		if r.Method != http.MethodGet {
			t.Errorf("machine_managers probe must be read-only, got %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"mm-1"}]`))
	})
	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	if err := (machineManagersCycle{}).Run(context.Background(), c, r); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	ops := r.Recorder.Ops()
	if len(ops) != 2 ||
		ops[0].Endpoint != "compute.openiaas.run_identity" ||
		ops[1].Endpoint != "compute.openiaas.machine_managers.list" {
		t.Fatalf("expected exactly [run_identity, machine_managers.list], got %+v", ops)
	}
	for _, o := range ops {
		if !o.OK || o.Skipped {
			t.Fatalf("op %s must be OK on a 200, got %+v", o.Endpoint, o)
		}
	}
	if r.Cleanup.Pending() != 0 {
		t.Fatalf("a read-only probe must register no teardown, got %d", r.Cleanup.Pending())
	}
}

// TestMachineManagersCycleSurfaces5xx pins that an http 5xx on the list is recorded
// as a FAILURE (the #315 signal the probe exists to catch), not swallowed. Mutation:
// ignore the List error → the op would read OK → RED.
func TestMachineManagersCycleSurfaces5xx(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // the transient ComputeManager flake
	})
	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	_ = (machineManagersCycle{}).Run(context.Background(), c, r)

	var found bool
	for _, o := range r.Recorder.Ops() {
		if o.Endpoint == "compute.openiaas.machine_managers.list" {
			found = true
			if o.OK || o.Skipped {
				t.Fatalf("machine_managers.list must FAIL on a 5xx, got %+v", o)
			}
		}
	}
	if !found {
		t.Fatal("machine_managers.list op must be recorded")
	}
}

// TestRegistryHasMachineManagersReadOnly pins that the cycle is registered and is a
// read-only cycle selectable WITHOUT -write (unlike the gated write lifecycle).
func TestRegistryHasMachineManagersReadOnly(t *testing.T) {
	reg := buildRegistry()
	sel, _, err := reg.Select("machine_managers", false)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(sel) != 1 || sel[0].Name() != "machine_managers" || sel[0].Kind() != KindRead {
		t.Fatalf("machine_managers must be a read-only selectable cycle, got %v", sel)
	}
}
