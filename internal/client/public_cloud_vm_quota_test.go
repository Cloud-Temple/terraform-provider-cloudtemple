package client

import (
	"context"
	"net/http"
	"testing"
)

// TestPublicCloudVMQuotaRead pins the singleton decode and that a 404 is surfaced
// as an ERROR (worker-not-found configuration error), never mapped to absence.
func TestPublicCloudVMQuotaRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 decodes limits and usage", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/quotas" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"vcpuLimit":100,"ramLimitMb":204800,"storageLimitGb":500,"vcpuUsed":7,"ramUsedMb":10240,"storageUsedGb":148}`))
		})
		q, err := c.PublicCloudVM().Quota().Read(ctx)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if q.VcpuLimit != 100 || q.RamLimitMb != 204800 || q.StorageLimitGb != 500 {
			t.Fatalf("limits not decoded: %+v", q)
		}
		if q.VcpuUsed != 7 || q.RamUsedMb != 10240 || q.StorageUsedGb != 148 {
			t.Fatalf("usage not decoded: %+v", q)
		}
	})

	t.Run("404 is an error (config), not absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		q, err := c.PublicCloudVM().Quota().Read(ctx)
		if err == nil {
			t.Fatalf("404 must surface as an error, got quota %+v", q)
		}
	})
}
