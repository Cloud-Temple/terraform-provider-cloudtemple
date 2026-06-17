package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// TestOpenIaaSVirtualMachineClientRelocate pins the relocate wiring (#355):
// POST /compute/v1/open_iaas/virtual_machines/{id}/relocate, body {hostId},
// and the activity id read back from the Location header.
func TestOpenIaaSVirtualMachineClientRelocate(t *testing.T) {
	ctx := context.Background()

	var gotMethod, gotPath, gotHostID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		var payload struct {
			HostId string `json:"hostId"`
		}
		_ = json.Unmarshal(body, &payload)
		gotHostID = payload.HostId
		w.Header().Set("Location", "activity-relocate-123")
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(&Config{Address: srv.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SavedToken = &jwt.Token{
		Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())},
	}

	activityID, err := c.Compute().OpenIaaS().VirtualMachine().Relocate(ctx, "vm-abc", &RelocateOpenIaasVirtualMachineRequest{HostId: "host-b"})
	if err != nil {
		t.Fatalf("Relocate: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/compute/v1/open_iaas/virtual_machines/vm-abc/relocate" {
		t.Errorf("path = %q, want /compute/v1/open_iaas/virtual_machines/vm-abc/relocate", gotPath)
	}
	if gotHostID != "host-b" {
		t.Errorf("body hostId = %q, want host-b", gotHostID)
	}
	if activityID != "activity-relocate-123" {
		t.Errorf("activityID = %q, want activity-relocate-123 (from Location header)", activityID)
	}
}
