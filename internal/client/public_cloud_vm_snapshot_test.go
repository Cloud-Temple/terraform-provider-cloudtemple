package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestPublicCloudVMSnapshotList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/snapshots" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total":1,"snapshots":[{"id":"s1","vmId":"vm-1","tenantId":"t1","name":"snap","status":"available","createdAt":"2026-07-01T12:00:00"}]}`))
	})
	snaps, err := c.PublicCloudVM().Snapshot().List(ctx, "vm-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(snaps) != 1 || snaps[0].ID != "s1" || snaps[0].VmID != "vm-1" || snaps[0].Name != "snap" || snaps[0].Status != "available" {
		t.Fatalf("snapshot not decoded: %+v", snaps)
	}
}

func TestPublicCloudVMSnapshotListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("truncated 200 (total > len) fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total":3,"snapshots":[{"id":"s1"}]}`))
		})
		if _, err := c.PublicCloudVM().Snapshot().ListStrict(ctx, "vm-1"); err == nil {
			t.Fatal("a truncated snapshot listing must fail closed")
		}
	})

	t.Run("206 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte(`{"total":1,"snapshots":[{"id":"s1"}]}`))
		})
		if _, err := c.PublicCloudVM().Snapshot().ListStrict(ctx, "vm-1"); err == nil {
			t.Fatal("a 206 must fail closed")
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		if _, err := c.PublicCloudVM().Snapshot().ListStrict(ctx, "vm-1"); err == nil {
			t.Fatal("a 403 must fail closed")
		}
	})

	t.Run("missing total (malformed wrapper) fails closed", func(t *testing.T) {
		for _, bad := range []string{`{"snapshots":[]}`, `{}`} {
			c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(bad))
			})
			if _, err := c.PublicCloudVM().Snapshot().ListStrict(ctx, "vm-1"); err == nil {
				t.Fatalf("a listing without a total (%s) must fail closed", bad)
			}
		}
	})

	t.Run("genuine empty listing (total 0 present) is accepted", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total":0,"snapshots":[]}`))
		})
		snaps, err := c.PublicCloudVM().Snapshot().ListStrict(ctx, "vm-1")
		if err != nil || len(snaps) != 0 {
			t.Fatalf("a genuine empty listing (total 0) must be accepted, got err=%v snaps=%d", err, len(snaps))
		}
	})

	t.Run("complete 200 returns the snapshots", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"total":1,"snapshots":[{"id":"s1"}]}`))
		})
		snaps, err := c.PublicCloudVM().Snapshot().ListStrict(ctx, "vm-1")
		if err != nil || len(snaps) != 1 {
			t.Fatalf("complete listing: err=%v snaps=%d", err, len(snaps))
		}
	})
}

func TestPublicCloudVMSnapshotRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/snapshots/s1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"s1","vmId":"vm-1","name":"snap","status":"available"}`))
		})
		snap, err := c.PublicCloudVM().Snapshot().Read(ctx, "vm-1", "s1")
		if err != nil || snap == nil || snap.Name != "snap" {
			t.Fatalf("bad snapshot: %+v err=%v", snap, err)
		}
	})

	t.Run("404 is absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
		snap, err := c.PublicCloudVM().Snapshot().Read(ctx, "vm-1", "missing")
		if err != nil || snap != nil {
			t.Fatalf("404 must be (nil,nil), got snap=%+v err=%v", snap, err)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		if _, err := c.PublicCloudVM().Snapshot().Read(ctx, "vm-1", "denied"); err == nil {
			t.Fatal("403 must fail closed")
		}
	})
}

func TestPublicCloudVMSnapshotWrites(t *testing.T) {
	ctx := context.Background()

	t.Run("create posts {name} and returns activityId", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/snapshots" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-snap")
			w.WriteHeader(http.StatusCreated)
		})
		id, err := c.PublicCloudVM().Snapshot().Create(ctx, "vm-1", "snap")
		if err != nil || id != "act-snap" {
			t.Fatalf("Create: id=%q err=%v", id, err)
		}
		if body["name"] != "snap" {
			t.Fatalf("name not sent: %v", body)
		}
	})

	t.Run("delete DELETEs the snapshot and returns activityId", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/snapshots/s1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Location", "act-snap-del")
			w.WriteHeader(http.StatusCreated)
		})
		id, err := c.PublicCloudVM().Snapshot().Delete(ctx, "vm-1", "s1")
		if err != nil || id != "act-snap-del" {
			t.Fatalf("Delete: id=%q err=%v", id, err)
		}
	})
}
