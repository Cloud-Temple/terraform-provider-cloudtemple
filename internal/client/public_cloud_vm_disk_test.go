package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestPublicCloudVMDiskList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/disks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"vmId":"vm-1","total":2,"disks":[
		  {"id":"d0","position":0,"label":"system","sizeGb":38,"storageType":"st-1","isPrimary":true},
		  {"id":"d1","position":1,"label":"data","sizeGb":10,"storageType":"st-1","isPrimary":false}
		]}`))
	})
	disks, err := c.PublicCloudVM().Disk().List(ctx, "vm-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(disks) != 2 {
		t.Fatalf("want 2 disks, got %d", len(disks))
	}
	if disks[0].Position != 0 || !disks[0].IsPrimary || disks[0].SizeGb != 38 {
		t.Fatalf("system disk not decoded: %+v", disks[0])
	}
	if disks[1].ID != "d1" || disks[1].Position != 1 || disks[1].IsPrimary || disks[1].StorageType != "st-1" {
		t.Fatalf("data disk not decoded: %+v", disks[1])
	}
}

// TestPublicCloudVMDiskListStrict pins the E0-9 completeness contract: only a
// complete 200 listing is absence evidence. A truncated 200 (total > len), a 206
// and a 403 all fail closed.
func TestPublicCloudVMDiskListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("complete 200 returns the disks", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"vmId":"vm-1","total":1,"disks":[{"id":"d1","position":1}]}`))
		})
		disks, err := c.PublicCloudVM().Disk().ListStrict(ctx, "vm-1")
		if err != nil || len(disks) != 1 {
			t.Fatalf("complete listing: err=%v disks=%d", err, len(disks))
		}
	})

	t.Run("truncated 200 (total > len) fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"vmId":"vm-1","total":5,"disks":[{"id":"d1","position":1}]}`))
		})
		if _, err := c.PublicCloudVM().Disk().ListStrict(ctx, "vm-1"); err == nil {
			t.Fatal("a truncated listing (total>len) must fail closed, not be usable as absence evidence")
		}
	})

	t.Run("missing total (malformed/empty wrapper) fails closed", func(t *testing.T) {
		// {"disks":[]} with no `total` must NOT be read as an authoritative empty
		// listing: a missing total decodes to 0 and len(nil)==0 would spuriously
		// prove absence. Kills the int-Total mutant.
		for _, bad := range []string{`{"disks":[]}`, `{}`} {
			c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(bad))
			})
			if _, err := c.PublicCloudVM().Disk().ListStrict(ctx, "vm-1"); err == nil {
				t.Fatalf("a listing without a total (%s) must fail closed", bad)
			}
		}
	})

	t.Run("genuine empty listing (total 0 present) is accepted", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"vmId":"vm-1","total":0,"disks":[]}`))
		})
		disks, err := c.PublicCloudVM().Disk().ListStrict(ctx, "vm-1")
		if err != nil || len(disks) != 0 {
			t.Fatalf("a genuine empty listing (total 0) must be accepted, got err=%v disks=%d", err, len(disks))
		}
	})

	t.Run("206 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte(`{"vmId":"vm-1","total":1,"disks":[{"id":"d1"}]}`))
		})
		if _, err := c.PublicCloudVM().Disk().ListStrict(ctx, "vm-1"); err == nil {
			t.Fatal("a 206 must fail closed")
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		if _, err := c.PublicCloudVM().Disk().ListStrict(ctx, "vm-1"); err == nil {
			t.Fatal("a 403 must fail closed")
		}
	})
}

func TestPublicCloudVMDiskRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/disks/d1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"d1","position":1,"label":"data","sizeGb":12,"storageType":"st-1","isPrimary":false}`))
		})
		disk, err := c.PublicCloudVM().Disk().Read(ctx, "vm-1", "d1")
		if err != nil || disk == nil || disk.SizeGb != 12 || disk.IsPrimary {
			t.Fatalf("bad disk: %+v err=%v", disk, err)
		}
	})

	t.Run("404 is absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
		disk, err := c.PublicCloudVM().Disk().Read(ctx, "vm-1", "missing")
		if err != nil || disk != nil {
			t.Fatalf("404 must be (nil,nil), got disk=%+v err=%v", disk, err)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		if _, err := c.PublicCloudVM().Disk().Read(ctx, "vm-1", "denied"); err == nil {
			t.Fatal("403 must fail closed")
		}
	})
}

func TestPublicCloudVMDiskWrites(t *testing.T) {
	ctx := context.Background()

	t.Run("create encodes camelCase body, omits empty storageType/name, returns activityId", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/disks" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-disk")
			w.WriteHeader(http.StatusCreated)
		})
		id, err := c.PublicCloudVM().Disk().Create(ctx, "vm-1", &CreateVMDiskRequest{Size: 10})
		if err != nil || id != "act-disk" {
			t.Fatalf("Create: id=%q err=%v", id, err)
		}
		if body["size"] != float64(10) {
			t.Fatalf("size not sent: %v", body)
		}
		if _, ok := body["storageType"]; ok {
			t.Fatalf("empty storageType must be omitted: %v", body)
		}
		if _, ok := body["name"]; ok {
			t.Fatalf("empty name must be omitted: %v", body)
		}
	})

	t.Run("extend posts {size} to /extend and returns activityId", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/disks/d1/extend" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-extend")
			w.WriteHeader(http.StatusCreated)
		})
		id, err := c.PublicCloudVM().Disk().ExtendById(ctx, "vm-1", "d1", 20)
		if err != nil || id != "act-extend" {
			t.Fatalf("ExtendById: id=%q err=%v", id, err)
		}
		if body["size"] != float64(20) {
			t.Fatalf("extend size not sent: %v", body)
		}
	})

	t.Run("delete DELETEs the disk and returns activityId", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/disks/d1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Location", "act-del")
			w.WriteHeader(http.StatusCreated)
		})
		id, err := c.PublicCloudVM().Disk().Delete(ctx, "vm-1", "d1")
		if err != nil || id != "act-del" {
			t.Fatalf("Delete: id=%q err=%v", id, err)
		}
	})
}
