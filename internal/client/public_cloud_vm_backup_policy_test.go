package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMBackupPolicyList pins the decode of the bare-array response
// including retention and the camelCase schedule fields.
func TestPublicCloudVMBackupPolicyList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/backup_policies" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"bp-1","name":"Daily backup — 30 days","description":"Daily incremental","retention":30,"scheduleCron":"0 20 * * *","scheduleWindowStartHour":20,"scheduleWindowDurationMinutes":240}]`))
	})
	pols, err := c.PublicCloudVM().BackupPolicy().List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(pols) != 1 {
		t.Fatalf("want 1 policy, got %d", len(pols))
	}
	p := pols[0]
	if p.ID != "bp-1" || p.Name != "Daily backup — 30 days" {
		t.Fatalf("strings not decoded: %+v", p)
	}
	if p.Retention != 30 || p.ScheduleCron != "0 20 * * *" || p.ScheduleWindowStartHour != 20 || p.ScheduleWindowDurationMinutes != 240 {
		t.Fatalf("schedule fields not decoded: %+v", p)
	}
}
