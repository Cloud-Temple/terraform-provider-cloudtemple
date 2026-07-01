package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMAvailabilityZoneList pins the decode of the bare-array,
// camelCase response including the nested compatibleFamilies and the regionId
// query-param wiring of the filter.
func TestPublicCloudVMAvailabilityZoneList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/availability_zones" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("regionId"); got != "reg-1" {
			t.Errorf("filter regionId not wired to query: got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name":"fr1-az01","description":null,"id":"az-1","regionId":"reg-1","isEnabled":true,"compatibleFamilies":[{"id":"fam-1","name":"Development"},{"id":"fam-2","name":"General Purpose"}],"createdAt":"2026-04-14T12:51:04Z","updatedAt":"2026-04-14T12:51:04Z"}]`))
	})

	zones, err := c.PublicCloudVM().AvailabilityZone().List(ctx, &PublicCloudVMAvailabilityZoneFilter{RegionID: "reg-1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(zones) != 1 {
		t.Fatalf("want 1 zone, got %d", len(zones))
	}
	z := zones[0]
	if z.ID != "az-1" || z.Name != "fr1-az01" || z.RegionID != "reg-1" || !z.IsEnabled {
		t.Fatalf("scalar fields not decoded: %+v", z)
	}
	if z.Description != "" {
		t.Fatalf("null description should decode to empty string, got %q", z.Description)
	}
	if len(z.CompatibleFamilies) != 2 || z.CompatibleFamilies[0].ID != "fam-1" || z.CompatibleFamilies[1].Name != "General Purpose" {
		t.Fatalf("compatibleFamilies not decoded: %+v", z.CompatibleFamilies)
	}
}

// TestPublicCloudVMAvailabilityZoneRead pins the fail-closed contract: 404 is
// absence (nil,nil), 403 (or other non-OK) errors and is never absence.
func TestPublicCloudVMAvailabilityZoneRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 by id decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/availability_zones/az-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"az-1","name":"fr1-az01","regionId":"reg-1","isEnabled":true}`))
		})
		az, err := c.PublicCloudVM().AvailabilityZone().Read(ctx, "az-1")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if az == nil || az.ID != "az-1" || az.RegionID != "reg-1" {
			t.Fatalf("bad az: %+v", az)
		}
	})

	t.Run("404 is absence (nil,nil)", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		az, err := c.PublicCloudVM().AvailabilityZone().Read(ctx, "missing")
		if err != nil {
			t.Fatalf("404 must not error, got %v", err)
		}
		if az != nil {
			t.Fatalf("404 must return nil, got %+v", az)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		az, err := c.PublicCloudVM().AvailabilityZone().Read(ctx, "denied")
		if err == nil {
			t.Fatalf("403 must fail closed with an error, got %+v", az)
		}
		if az != nil {
			t.Fatalf("403 must not return an az")
		}
	})
}
