package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

// These unit tests pin the #382 fix: a datasource read whose client method maps
// an HTTP 404/403 to a (nil, nil) result must surface an actionable diagnostic,
// never a nil-pointer panic. Each test drives the real datasource read function
// against a stub HTTP server (newAssignTestClient) and asserts a clean error.
//
// Mutation proof: removing the corresponding `if x == nil` guard makes the test
// go RED via a nil-pointer dereference (d.SetId / Flatten* on a nil pointer).

// TestDataSourceBucketReadNilDoesNotPanic: a 404 (absent) or 403 (forbidden) read
// of a bucket by name must return a diagnostic, not panic at d.SetId(bucket.ID).
func TestDataSourceBucketReadNilDoesNotPanic(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusForbidden} {
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

// TestDataSourceStorageAccountReadNilDoesNotPanic: same contract for the storage
// account datasource (d.SetId(account.ID)).
func TestDataSourceStorageAccountReadNilDoesNotPanic(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusForbidden} {
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

// backupMetricsHandler serves a valid 200 body for every backup-metrics endpoint
// EXCEPT the one whose path is in `down`, which returns `downCode`. Returning a
// valid 200 for every other endpoint is what makes the mutation proof real: if a
// guard is removed, the read must reach the Flatten*(nil) call rather than
// failing early on an unrelated read (the read order is Coverage, History,
// Platform, PlatformCPU, Policies, VirtualMachines).
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

// TestBackupMetricsReadNilHistoryDoesNotPanic: the History endpoint returns 404
// (mapped to (nil,nil)) while every other endpoint returns a valid 200. The read
// must surface a diagnostic instead of panicking in FlattenBackupMetricsHistory.
func TestBackupMetricsReadNilHistoryDoesNotPanic(t *testing.T) {
	c := newAssignTestClient(t, backupMetricsHandler(t, "/backup/v1/spp/metrics/backup/history", http.StatusNotFound))
	d := dataSourceBackupMetrics().TestResourceData()
	_ = d.Set("range", 7)

	diags := backupMetricsRead(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a nil history read must return an error diagnostic, got none")
	}
	diagsContain(t, diags, "history is unavailable")
}

// TestBackupMetricsReadNilPlatformCPUDoesNotPanic: the PlatformCPU endpoint
// returns 403 (mapped to (nil,nil)) while every other endpoint returns a valid
// 200. The read must surface a diagnostic instead of panicking in
// FlattenBackupMetricsPlatformCPU.
func TestBackupMetricsReadNilPlatformCPUDoesNotPanic(t *testing.T) {
	c := newAssignTestClient(t, backupMetricsHandler(t, "/backup/v1/spp/metrics/plateform/cpu", http.StatusForbidden))
	d := dataSourceBackupMetrics().TestResourceData()
	_ = d.Set("range", 7)

	diags := backupMetricsRead(context.Background(), d, c)
	if !diags.HasError() {
		t.Fatal("a nil platform CPU read must return an error diagnostic, got none")
	}
	diagsContain(t, diags, "platform CPU is unavailable")
}
