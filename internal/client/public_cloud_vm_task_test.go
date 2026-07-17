package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMTaskList pins the two list paths (global vs per-VM) and the
// limit query param.
func TestPublicCloudVMTaskList(t *testing.T) {
	ctx := context.Background()

	t.Run("global list with limit", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/tasks" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("limit"); got != "5" {
				t.Errorf("limit not wired: %q", got)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"task-1","vmId":"vm-1","taskType":"vm_restore","status":"success","message":"ok","failureCode":null,"createdAt":"2026-07-01T08:56:42Z","completedAt":"2026-07-01T08:58:24Z"}]`))
		})
		tasks, err := c.PublicCloudVM().Task().List(ctx, "", 5)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(tasks) != 1 || tasks[0].ID != "task-1" || tasks[0].VmID != "vm-1" || tasks[0].TaskType != "vm_restore" || tasks[0].Status != "success" {
			t.Fatalf("task not decoded: %+v", tasks)
		}
	})

	t.Run("per-VM list uses the VM path", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/tasks" {
				t.Errorf("expected per-VM path, got: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		})
		if _, err := c.PublicCloudVM().Task().List(ctx, "vm-1", 0); err != nil {
			t.Fatalf("List per-VM: %v", err)
		}
	})
}

// TestPublicCloudVMTaskRead pins the fail-closed contract: 404=absence, 403=error.
func TestPublicCloudVMTaskRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 by id", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/tasks/task-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"task-1","status":"running"}`))
		})
		task, err := c.PublicCloudVM().Task().Read(ctx, "task-1")
		if err != nil || task == nil || task.ID != "task-1" {
			t.Fatalf("bad read: task=%+v err=%v", task, err)
		}
	})

	t.Run("404 is absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
		task, err := c.PublicCloudVM().Task().Read(ctx, "missing")
		if err != nil || task != nil {
			t.Fatalf("404 should be (nil,nil): task=%+v err=%v", task, err)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		task, err := c.PublicCloudVM().Task().Read(ctx, "denied")
		if err == nil || task != nil {
			t.Fatalf("403 must fail closed: task=%+v err=%v", task, err)
		}
	})
}
