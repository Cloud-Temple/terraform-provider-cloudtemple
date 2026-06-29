package client

import (
	"context"
	"net/http"
	"testing"
)

// TestProbeStatus pins that ProbeStatus returns the RAW HTTP status for a GET,
// without the not-found/forbidden folding the resource read methods apply. The
// distinction between 404 (migrated) and 403 (forbidden) is exactly what the
// absence-contract probe needs and what requireNotFoundOrOK would have erased.
//
// Only non-retried statuses are exercised here (2xx/4xx): a 5xx would trigger the
// client's transient-retry path; the probe's handling of 5xx is covered by the
// ct-validate probeAbsenceOutcome unit test.
func TestProbeStatus(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusForbidden, http.StatusNotFound, http.StatusConflict} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			c := newVPCTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(code)
			})
			got, err := c.ProbeStatus(context.Background(), "/compute/v1/vcenters/virtual_machines/%s", "bogus-id")
			if err != nil {
				t.Fatalf("ProbeStatus returned an unexpected error for HTTP %d: %v", code, err)
			}
			if got != code {
				t.Fatalf("ProbeStatus: expected raw status %d, got %d", code, got)
			}
		})
	}
}
