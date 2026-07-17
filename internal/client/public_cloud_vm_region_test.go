package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMRegionList pins the decode of the bare-array, camelCase
// /vm_instances/v1/regions response, including a null `description` -> "".
func TestPublicCloudVMRegionList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/regions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name":"fr1","description":null,"countryCode":"FR","geography":"Europe","id":"reg-1","isEnabled":true,"azCount":2,"createdAt":"2026-04-14T12:50:41.881814","updatedAt":"2026-04-14T12:50:41.881814"}]`))
	})

	regions, err := c.PublicCloudVM().Region().List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("want 1 region, got %d", len(regions))
	}
	got := regions[0]
	if got.ID != "reg-1" || got.Name != "fr1" || got.CountryCode != "FR" || got.Geography != "Europe" {
		t.Fatalf("string fields not decoded: %+v", got)
	}
	if !got.IsEnabled {
		t.Fatalf("isEnabled should decode to true")
	}
	if got.AzCount != 2 {
		t.Fatalf("azCount want 2, got %d", got.AzCount)
	}
	if got.Description != "" {
		t.Fatalf("null description should decode to empty string, got %q", got.Description)
	}
	if got.CreatedAt == "" || got.UpdatedAt == "" {
		t.Fatalf("timestamps not decoded: %+v", got)
	}
}

// TestPublicCloudVMRegionRead pins the state-safety contract of Read: a positive
// 404 is absence (nil,nil), but a 403 (or any other non-OK) fails closed with an
// error and never maps to absence. Mutating Read to requireNotFoundOrOK(resp,403)
// would turn the 403 sub-test RED.
func TestPublicCloudVMRegionRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 by id decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/regions/reg-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"reg-1","name":"fr1","countryCode":"FR","isEnabled":true,"azCount":2}`))
		})
		reg, err := c.PublicCloudVM().Region().Read(ctx, "reg-1")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if reg == nil || reg.ID != "reg-1" || reg.Name != "fr1" {
			t.Fatalf("bad region: %+v", reg)
		}
	})

	t.Run("404 is absence (nil,nil)", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		reg, err := c.PublicCloudVM().Region().Read(ctx, "missing")
		if err != nil {
			t.Fatalf("404 must not error, got %v", err)
		}
		if reg != nil {
			t.Fatalf("404 must return a nil region, got %+v", reg)
		}
	})

	t.Run("403 fails closed (error, not absence)", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		reg, err := c.PublicCloudVM().Region().Read(ctx, "denied")
		if err == nil {
			t.Fatalf("403 must fail closed with an error, got region %+v", reg)
		}
		if reg != nil {
			t.Fatalf("403 must not return a region")
		}
	})
}
