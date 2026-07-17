package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMTemplateList pins the decode of the bare-array response
// including the int/string slice fields and the two filter query params.
func TestPublicCloudVMTemplateList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/templates" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("instanceFamilyId"); got != "fam-1" {
			t.Errorf("instanceFamilyId filter not wired: %q", got)
		}
		if got := r.URL.Query().Get("availabilityZoneId"); got != "az-1" {
			t.Errorf("availabilityZoneId filter not wired: %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"tpl-1","name":"Rocky Linux 9","osFamily":"linux","osName":"Rocky Linux 9","osVersion":"9","diskSizesGb":[38,50],"compatibleFamilies":["fam-1","fam-2"],"icon":"data:image/png;base64,AAAA"}]`))
	})
	tpls, err := c.PublicCloudVM().Template().List(ctx, &PublicCloudVMTemplateFilter{InstanceFamilyID: "fam-1", AvailabilityZoneID: "az-1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tpls) != 1 {
		t.Fatalf("want 1 template, got %d", len(tpls))
	}
	tpl := tpls[0]
	if tpl.ID != "tpl-1" || tpl.OsFamily != "linux" || tpl.OsName != "Rocky Linux 9" || tpl.OsVersion != "9" {
		t.Fatalf("os fields not decoded: %+v", tpl)
	}
	if len(tpl.DiskSizesGb) != 2 || tpl.DiskSizesGb[0] != 38 || tpl.DiskSizesGb[1] != 50 {
		t.Fatalf("disk_sizes_gb not decoded: %+v", tpl.DiskSizesGb)
	}
	if len(tpl.CompatibleFamilies) != 2 || tpl.CompatibleFamilies[0] != "fam-1" {
		t.Fatalf("compatible_families not decoded: %+v", tpl.CompatibleFamilies)
	}
	if tpl.Icon == "" {
		t.Fatalf("icon not decoded")
	}
}

// TestPublicCloudVMTemplateRead pins the fail-closed contract: 404=absence, 403=error.
func TestPublicCloudVMTemplateRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 by id", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/templates/tpl-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"tpl-1","name":"Rocky Linux 9","osFamily":"linux"}`))
		})
		tpl, err := c.PublicCloudVM().Template().Read(ctx, "tpl-1")
		if err != nil || tpl == nil || tpl.ID != "tpl-1" {
			t.Fatalf("bad read: tpl=%+v err=%v", tpl, err)
		}
	})

	t.Run("404 is absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
		tpl, err := c.PublicCloudVM().Template().Read(ctx, "missing")
		if err != nil || tpl != nil {
			t.Fatalf("404 should be (nil,nil): tpl=%+v err=%v", tpl, err)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		tpl, err := c.PublicCloudVM().Template().Read(ctx, "denied")
		if err == nil || tpl != nil {
			t.Fatalf("403 must fail closed: tpl=%+v err=%v", tpl, err)
		}
	})
}
