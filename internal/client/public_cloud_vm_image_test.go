package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMImageList pins the decode of the bare-array response
// including the int/string slice fields, the imageType field (case-insensitive
// match, no json tag) and the two filter query params. It also pins the request
// path to /vm_instances/v1/images so a revert to /templates turns this RED.
func TestPublicCloudVMImageList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/images" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("instanceFamilyId"); got != "fam-1" {
			t.Errorf("instanceFamilyId filter not wired: %q", got)
		}
		if got := r.URL.Query().Get("availabilityZoneId"); got != "az-1" {
			t.Errorf("availabilityZoneId filter not wired: %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"img-1","name":"Rocky Linux 9","osFamily":"linux","osName":"Rocky Linux 9","osVersion":"9","diskSizesGb":[38,50],"compatibleFamilies":["fam-1","fam-2"],"imageType":"os","icon":"data:image/png;base64,AAAA"}]`))
	})
	imgs, err := c.PublicCloudVM().Image().List(ctx, &PublicCloudVMImageFilter{InstanceFamilyID: "fam-1", AvailabilityZoneID: "az-1"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(imgs) != 1 {
		t.Fatalf("want 1 image, got %d", len(imgs))
	}
	img := imgs[0]
	if img.ID != "img-1" || img.OsFamily != "linux" || img.OsName != "Rocky Linux 9" || img.OsVersion != "9" {
		t.Fatalf("os fields not decoded: %+v", img)
	}
	if len(img.DiskSizesGb) != 2 || img.DiskSizesGb[0] != 38 || img.DiskSizesGb[1] != 50 {
		t.Fatalf("disk_sizes_gb not decoded: %+v", img.DiskSizesGb)
	}
	if len(img.CompatibleFamilies) != 2 || img.CompatibleFamilies[0] != "fam-1" {
		t.Fatalf("compatible_families not decoded: %+v", img.CompatibleFamilies)
	}
	if img.ImageType != "os" {
		t.Fatalf("image_type (imageType) not decoded: %q", img.ImageType)
	}
	if img.Icon == "" {
		t.Fatalf("icon not decoded")
	}
}

// TestPublicCloudVMImageRead pins the fail-closed contract: 404=absence, 403=error,
// and the request path /vm_instances/v1/images/{id}.
func TestPublicCloudVMImageRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 by id", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/images/img-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"img-1","name":"Rocky Linux 9","osFamily":"linux"}`))
		})
		img, err := c.PublicCloudVM().Image().Read(ctx, "img-1")
		if err != nil || img == nil || img.ID != "img-1" {
			t.Fatalf("bad read: img=%+v err=%v", img, err)
		}
	})

	t.Run("404 is absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
		img, err := c.PublicCloudVM().Image().Read(ctx, "missing")
		if err != nil || img != nil {
			t.Fatalf("404 should be (nil,nil): img=%+v err=%v", img, err)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		img, err := c.PublicCloudVM().Image().Read(ctx, "denied")
		if err == nil || img != nil {
			t.Fatalf("403 must fail closed: img=%+v err=%v", img, err)
		}
	})
}
