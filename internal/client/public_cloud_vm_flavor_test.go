package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMFlavorList pins the WRAPPED-response decode ({ "flavors": [...] },
// unlike the bare-array region/az endpoints) and the familyId filter wiring.
func TestPublicCloudVMFlavorList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/flavors" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("familyId"); got != "fam-1" {
			t.Errorf("filter familyId not wired to query: got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"flavors":[{"id":"flv-1","instanceFamilyId":"fam-1","name":"dev-micro","vcpu":1,"ramGb":2},{"id":"flv-2","instanceFamilyId":"fam-1","name":"dev-small","vcpu":2,"ramGb":4}]}`))
	})

	flavors, err := c.PublicCloudVM().Flavor().List(ctx, &PublicCloudVMFlavorFilter{FamilyID: "fam-1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(flavors) != 2 {
		t.Fatalf("want 2 flavors, got %d", len(flavors))
	}
	f := flavors[0]
	if f.ID != "flv-1" || f.InstanceFamilyID != "fam-1" || f.Name != "dev-micro" || f.Vcpu != 1 || f.RamGb != 2 {
		t.Fatalf("flavor not decoded from wrapped response: %+v", f)
	}
}

// TestPublicCloudVMFlavorListEmptyWrapper pins that a wrapper without the flavors
// key (or empty) yields an empty slice, not a decode error.
func TestPublicCloudVMFlavorListEmptyWrapper(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"flavors":[]}`))
	})
	flavors, err := c.PublicCloudVM().Flavor().List(ctx, &PublicCloudVMFlavorFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(flavors) != 0 {
		t.Fatalf("want 0 flavors, got %d", len(flavors))
	}
}
