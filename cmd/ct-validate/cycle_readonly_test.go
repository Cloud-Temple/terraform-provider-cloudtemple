package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/golang-jwt/jwt/v4"
)

// newReadonlyTestClient wires a Client to a stub HTTP server with a pre-seeded
// far-future JWT, so the cycle exercises the real request/decoding path without
// any auth network call.
func newReadonlyTestClient(t *testing.T, h http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{Address: srv.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SavedToken = &jwt.Token{Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())}}
	return c
}

func opsByEndpoint(r *Run) map[string]Op {
	m := map[string]Op{}
	for _, o := range r.Recorder.Ops() {
		m[o.Endpoint] = o
	}
	return m
}

// isVMwareVMList matches the VMware virtual-machines list (path
// /compute/v1/vcenters/virtual_machines), NOT the OpenIaaS one.
func isVMwareVMList(p string) bool {
	return strings.Contains(p, "/vcenters/virtual_machines")
}

// isOpenIaaSMachineManagers matches the OpenIaaS machine-managers list, whose
// path is the bare /compute/v1/open_iaas (no resource suffix).
func isOpenIaaSMachineManagers(p string) bool {
	return strings.HasSuffix(p, "/open_iaas")
}

// TestReadonlyComputeVMwareProbeSkips pins that a 4xx on the VMware probe
// (compute.virtual_machines.list) SKIPS the rest of the VMware block instead of
// emitting a failure per endpoint — a tenant without VMware must not produce
// seven false "squeaks". Mutation proof: remove the
// `categorize(vmwareErr)==CategoryHTTP4xx` skip branch in runCompute and those
// endpoints are ATTEMPTED (recorded as failures, not skips) → RED.
func TestReadonlyComputeVMwareProbeSkips(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case isVMwareVMList(p):
			w.WriteHeader(http.StatusBadRequest) // 4xx → probe-skip the rest
		case isOpenIaaSMachineManagers(p):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`)) // no machine manager → OpenIaaS scoped skipped too
		default:
			t.Errorf("unexpected call after probe-skip: %s %s", r.Method, p)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}
	})

	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	readonlyCycle{}.runCompute(context.Background(), c, r)

	ops := opsByEndpoint(r)
	for _, ep := range []string{
		"compute.virtual_machines.read", "compute.datastores.list", "compute.hosts.list",
		"compute.networks.list", "compute.virtual_datacenters.list", "compute.folders.list",
		"compute.virtual_disks.list",
	} {
		o, ok := ops[ep]
		if !ok || !o.Skipped {
			t.Fatalf("%s must be SKIPPED after the VMware probe 4xx, got %+v (present=%v)", ep, o, ok)
		}
	}
	if o := ops["compute.virtual_machines.list"]; o.Skipped || o.Category != CategoryHTTP4xx {
		t.Fatalf("the VMware probe must be recorded as a 4xx failure, got %+v", o)
	}
}

// TestReadonlyComputeOpenIaaSScoped pins that every OpenIaaS list is scoped by
// the discovered machine_manager_id (the API answers 5xx without it). Mutation
// proof: pass a nil filter (drop the MachineManagerID) on any of these calls and
// the "must carry machineManagerId=mm-1" assertion goes RED.
func TestReadonlyComputeOpenIaaSScoped(t *testing.T) {
	var mu sync.Mutex
	scopedSeen := map[string]string{}

	c := newReadonlyTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case isOpenIaaSMachineManagers(p):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"mm-1"}]`))
		case strings.Contains(p, "/open_iaas/"):
			mu.Lock()
			scopedSeen[p[strings.Index(p, "/open_iaas/"):]] = r.URL.Query().Get("machineManagerId")
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		default: // VMware block (probe ok-empty so it runs harmlessly) and anything else.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		}
	})

	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	readonlyCycle{}.runCompute(context.Background(), c, r)

	mu.Lock()
	defer mu.Unlock()
	for _, suffix := range []string{
		"/open_iaas/virtual_machines", "/open_iaas/networks", "/open_iaas/storage_repositories",
		"/open_iaas/templates", "/open_iaas/hosts", "/open_iaas/pools",
	} {
		if got := scopedSeen[suffix]; got != "mm-1" {
			t.Fatalf("OpenIaaS list %s must be scoped by machineManagerId=mm-1, got %q (seen=%v)", suffix, got, scopedSeen)
		}
	}
}

// TestReadonlyBackupProbeSkips pins the same probe-and-skip on the backup block:
// a 4xx on the entry list (backup.sla_policies.list) SKIPS the rest instead of
// emitting a squeak per endpoint (tenant without backup). Mutation proof: remove
// the `categorize(backupErr)==CategoryHTTP4xx` skip branch in runBackup and
// sites/storages/spp_servers are ATTEMPTED (recorded as failures) → RED.
func TestReadonlyBackupProbeSkips(t *testing.T) {
	c := newReadonlyTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // runBackup only hits backup endpoints; 4xx on the probe
	})
	r := &Run{Recorder: NewRecorder(), Breaker: NewBreaker(1000, 0.99, 1000), Cleanup: NewCleanup()}
	readonlyCycle{}.runBackup(context.Background(), c, r)

	ops := opsByEndpoint(r)
	if o := ops["backup.sla_policies.list"]; o.Skipped || o.Category != CategoryHTTP4xx {
		t.Fatalf("the backup probe must be recorded as a 4xx failure, got %+v", o)
	}
	for _, ep := range []string{"backup.sites.list", "backup.storages.list", "backup.spp_servers.list"} {
		o, ok := ops[ep]
		if !ok || !o.Skipped {
			t.Fatalf("%s must be SKIPPED after the backup probe 4xx, got %+v (present=%v)", ep, o, ok)
		}
	}
}
