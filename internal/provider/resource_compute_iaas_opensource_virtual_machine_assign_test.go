package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/golang-jwt/jwt/v4"
)

// newAssignTestClient wires a Client to a stub HTTP server with a pre-seeded
// far-future JWT, so JWT() never hits the network: these tests exercise the
// assign gating, not the auth plumbing.
func newAssignTestClient(t *testing.T, h http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{Address: srv.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SavedToken = &jwt.Token{
		Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())},
	}
	return c
}

// TestAssignBackupSLAPoliciesIfAny pins the #306 fix: an EMPTY SLA policy list
// must issue NO backup-assign call (an empty-list assign creates an activity
// that hangs platform-side and previously hung the VM Create until timeout). A
// NON-EMPTY list must still call the assign endpoint and wait for completion.
//
// Non-complacency: the empty case fails the stub server on any request, so it
// goes RED if the no-op gate is removed; the non-empty case asserts the assign
// endpoint IS hit, so it goes RED if the gate ever over-skips.
func TestAssignBackupSLAPoliciesIfAny(t *testing.T) {
	ctx := context.Background()

	t.Run("empty list issues zero HTTP calls", func(t *testing.T) {
		hits := 0
		c := newAssignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			hits++
			t.Errorf("no HTTP request expected for an empty SLA policy list, got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		})
		if err := assignBackupSLAPoliciesIfAny(ctx, c, "vm-1", []string{}); err != nil {
			t.Fatalf("empty policy list must be a no-op, got error: %v", err)
		}
		if hits != 0 {
			t.Fatalf("empty policy list must issue zero HTTP calls, got %d", hits)
		}
	})

	t.Run("non-empty list calls the assign endpoint and waits", func(t *testing.T) {
		var assignHit bool
		c := newAssignTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/backup/v1/open_iaas/policies/assign"):
				assignHit = true
				w.Header().Set("Location", "act-1")
				w.WriteHeader(http.StatusOK)
			case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/activity/v1/activities/act-1"):
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"act-1","state":{"completed":{}}}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		if err := assignBackupSLAPoliciesIfAny(ctx, c, "vm-1", []string{"pol-1"}); err != nil {
			t.Fatalf("non-empty policy list must succeed against a completed activity, got: %v", err)
		}
		if !assignHit {
			t.Fatal("non-empty policy list must call the backup assign endpoint")
		}
	})
}
