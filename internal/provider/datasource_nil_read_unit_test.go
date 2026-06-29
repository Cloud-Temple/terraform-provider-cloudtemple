package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// These unit tests pin the #382 fix: a datasource read whose client method maps
// an HTTP 404/403 to a (nil, nil) result must surface an actionable diagnostic,
// never a nil-pointer panic — AND a normal 200 read must keep working unchanged.
//
// Both the OLD not-found contract (absent -> 403) and the NEW one (absent -> 404)
// reach the client helper requireNotFoundOrOK(resp, 403), which folds 404 AND 403
// to (nil, nil); so each nil-read test is run under BOTH status codes to prove the
// guard behaves identically before and after the API change.
//
// Mutation proof (recorded): removing a guard makes the matching nil-read test go
// RED via a nil-pointer dereference (d.SetId / Flatten* on a nil pointer).

// absentCodes are the two HTTP statuses the client maps to (nil, nil): the new
// not-found (404) and the old not-found / current forbidden (403).
var absentCodes = []int{http.StatusNotFound, http.StatusForbidden}

// --- object storage: nil read must error (both 403 and 404), never panic ---

func TestDataSourceBucketReadNilDoesNotPanic(t *testing.T) {
	for _, code := range absentCodes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(code)
			})
			d := dataSourceBucket().TestResourceData()
			_ = d.Set("name", "missing-bucket")

			diags := dataSourceBucketRead(context.Background(), d, c)
			if !diags.HasError() {
				t.Fatalf("a %d bucket read must return an error diagnostic, got none", code)
			}
			diagsContain(t, diags, "could not be read")
		})
	}
}

func TestDataSourceStorageAccountReadNilDoesNotPanic(t *testing.T) {
	for _, code := range absentCodes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(code)
			})
			d := dataSourceStorageAccount().TestResourceData()
			_ = d.Set("name", "missing-account")

			diags := dataSourceStorageAccountRead(context.Background(), d, c)
			if !diags.HasError() {
				t.Fatalf("a %d storage account read must return an error diagnostic, got none", code)
			}
			diagsContain(t, diags, "could not be read")
		})
	}
}

// --- object storage: a normal 200 read must still succeed unchanged ---

func TestDataSourceBucketReadSuccess(t *testing.T) {
	c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"bucket-123","name":"my-bucket"}`))
	})
	d := dataSourceBucket().TestResourceData()
	_ = d.Set("name", "my-bucket")

	diags := dataSourceBucketRead(context.Background(), d, c)
	if diags.HasError() {
		t.Fatalf("a valid 200 bucket read must succeed, got: %v", diags)
	}
	if d.Id() != "bucket-123" {
		t.Fatalf("expected id %q, got %q", "bucket-123", d.Id())
	}
}

func TestDataSourceStorageAccountReadSuccess(t *testing.T) {
	c := newAssignTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"acc-1","name":"my-account"}`))
	})
	d := dataSourceStorageAccount().TestResourceData()
	_ = d.Set("name", "my-account")

	diags := dataSourceStorageAccountRead(context.Background(), d, c)
	if diags.HasError() {
		t.Fatalf("a valid 200 storage account read must succeed, got: %v", diags)
	}
	if d.Id() != "acc-1" {
		t.Fatalf("expected id %q, got %q", "acc-1", d.Id())
	}
}

// backupMetricsHandler serves a valid 200 body for every backup-metrics endpoint
// EXCEPT the one whose path is `downPath`, which returns `downCode`. Returning a
// valid 200 for every other endpoint is what makes the mutation proof real: if a
// guard is removed, the read must reach the Flatten*(nil) call rather than failing
// early on an unrelated read (read order: Coverage, History, Platform, PlatformCPU,
// Policies, VirtualMachines).
func backupMetricsHandler(t *testing.T, downPath string, downCode int) http.HandlerFunc {
	t.Helper()
	// "{}" decodes into the metric structs (zero values, non-nil pointer); the
	// policies endpoint returns a JSON array.
	bodies := map[string]string{
		"/backup/v1/spp/metrics/coverage":       "{}",
		"/backup/v1/spp/metrics/backup/history": "{}",
		"/backup/v1/spp/metrics/plateform":      "{}",
		"/backup/v1/spp/metrics/plateform/cpu":  "{}",
		"/backup/v1/spp/metrics/policies":       "[]",
		"/backup/v1/spp/metrics/vm":             "{}",
	}
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == downPath {
			w.WriteHeader(downCode)
			return
		}
		body, ok := bodies[p]
		if !ok {
			t.Errorf("unexpected backup-metrics request path: %s", p)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}
}

// --- backup metrics: a nil History / PlatformCPU read must error (both 403 and
// 404), never panic ---

func TestBackupMetricsReadNilHistoryDoesNotPanic(t *testing.T) {
	for _, code := range absentCodes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			c := newAssignTestClient(t, backupMetricsHandler(t, "/backup/v1/spp/metrics/backup/history", code))
			d := dataSourceBackupMetrics().TestResourceData()
			_ = d.Set("range", 7)

			diags := backupMetricsRead(context.Background(), d, c)
			if !diags.HasError() {
				t.Fatalf("a %d history read must return an error diagnostic, got none", code)
			}
			diagsContain(t, diags, "history is unavailable")
		})
	}
}

func TestBackupMetricsReadNilPlatformCPUDoesNotPanic(t *testing.T) {
	for _, code := range absentCodes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			c := newAssignTestClient(t, backupMetricsHandler(t, "/backup/v1/spp/metrics/plateform/cpu", code))
			d := dataSourceBackupMetrics().TestResourceData()
			_ = d.Set("range", 7)

			diags := backupMetricsRead(context.Background(), d, c)
			if !diags.HasError() {
				t.Fatalf("a %d platform CPU read must return an error diagnostic, got none", code)
			}
			diagsContain(t, diags, "platform CPU is unavailable")
		})
	}
}

// TestBackupMetricsReadSuccessPopulatesState proves the success path is unchanged
// AND that a legitimate ZERO value is NOT mistaken for a nil/absent read: every
// endpoint returns a valid 200, History carries a non-zero value (total_runs=10)
// and PlatformCPU carries an explicit zero (cpu_util=0). The read must succeed,
// set the id, flow the History value through, and keep the zero-valued
// platform_cpu block (non-nil), not drop or error on it.
func TestBackupMetricsReadSuccessPopulatesState(t *testing.T) {
	bodies := map[string]string{
		"/backup/v1/spp/metrics/coverage":       `{}`,
		"/backup/v1/spp/metrics/backup/history": `{"totalRuns":10,"success":8}`,
		"/backup/v1/spp/metrics/plateform":      `{}`,
		"/backup/v1/spp/metrics/plateform/cpu":  `{"cpuUtil":0}`,
		"/backup/v1/spp/metrics/policies":       `[]`,
		"/backup/v1/spp/metrics/vm":             `{}`,
	}
	c := newAssignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path]
		if !ok {
			t.Errorf("unexpected backup-metrics request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
	d := dataSourceBackupMetrics().TestResourceData()
	_ = d.Set("range", 7)

	diags := backupMetricsRead(context.Background(), d, c)
	if diags.HasError() {
		t.Fatalf("a full valid 200 metrics read must succeed, got: %v", diags)
	}
	if d.Id() != "job_sessions" {
		t.Fatalf("expected id %q, got %q", "job_sessions", d.Id())
	}
	// The non-zero History value must flow through unchanged (the guard did not
	// swallow real data).
	if got := d.Get("history.0.total_runs").(int); got != 10 {
		t.Fatalf("expected history.0.total_runs == 10, got %d", got)
	}
	// The zero-valued PlatformCPU is present (a valid 200 {} / cpu_util:0 decodes
	// to a NON-nil struct, so the guard must NOT fire): the block exists.
	if cpu, ok := d.Get("platform_cpu").([]interface{}); !ok || len(cpu) != 1 {
		t.Fatalf("expected one platform_cpu block (zero value must not be dropped), got %v", d.Get("platform_cpu"))
	}
}
