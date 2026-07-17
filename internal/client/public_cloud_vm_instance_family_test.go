package client

import (
	"context"
	"net/http"
	"testing"
)

func TestPublicCloudVMInstanceFamilyList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/instance_families" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"fam-1","name":"Development","description":"Development familly","vcpuMin":1,"vcpuMax":16,"ramMinGb":2,"ramMaxGb":24}]`))
	})
	fams, err := c.PublicCloudVM().InstanceFamily().List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(fams) != 1 {
		t.Fatalf("want 1 family, got %d", len(fams))
	}
	f := fams[0]
	if f.ID != "fam-1" || f.Name != "Development" || f.Description != "Development familly" {
		t.Fatalf("strings not decoded: %+v", f)
	}
	if f.VcpuMin != 1 || f.VcpuMax != 16 || f.RamMinGb != 2 || f.RamMaxGb != 24 {
		t.Fatalf("bounds not decoded: %+v", f)
	}
}

func TestPublicCloudVMInstanceFamilyRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 by id decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/instance_families/fam-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fam-1","name":"Development","vcpuMin":1,"vcpuMax":16}`))
		})
		f, err := c.PublicCloudVM().InstanceFamily().Read(ctx, "fam-1")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if f == nil || f.ID != "fam-1" || f.VcpuMax != 16 {
			t.Fatalf("bad family: %+v", f)
		}
	})

	t.Run("404 is absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
		f, err := c.PublicCloudVM().InstanceFamily().Read(ctx, "missing")
		if err != nil || f != nil {
			t.Fatalf("404 should be (nil,nil), got f=%+v err=%v", f, err)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		f, err := c.PublicCloudVM().InstanceFamily().Read(ctx, "denied")
		if err == nil || f != nil {
			t.Fatalf("403 must fail closed, got f=%+v err=%v", f, err)
		}
	})
}
