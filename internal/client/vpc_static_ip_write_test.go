package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

// TestVPCStaticIPCreate pins the SYNCHRONOUS create contract: the id comes from
// the 201 BODY (static_ip_id), never from a Location header; a missing/empty id
// or a non-201 status is an error; the request body carries the create fields.
func TestVPCStaticIPCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("201 parses static_ip_id from the BODY and posts the right path/body", func(t *testing.T) {
		var gotBody map[string]any
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vpc/v1/private_networks/pn-1/static_ips" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			// A Location header MUST be ignored: this is a sync create, the id is
			// in the body. Setting one here proves Create does not read it.
			w.Header().Set("Location", "should-be-ignored")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"success":true,"message":"ok","static_ip_id":"si-new"}`))
		})

		id, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			IPAddress:           "10.0.1.50",
			ResourceDescription: "web",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "si-new" {
			t.Fatalf("id must come from the body static_ip_id, got %q (a %q here means it read Location)", id, "should-be-ignored")
		}
		if gotBody["macAddress"] != "00:50:56:ab:cd:ef" || gotBody["ipAddress"] != "10.0.1.50" || gotBody["resourceDescription"] != "web" {
			t.Fatalf("unexpected create body: %v", gotBody)
		}
	})

	t.Run("optional fields omitted are not sent", func(t *testing.T) {
		var gotBody map[string]any
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"static_ip_id":"si-x"}`))
		})
		if _, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := gotBody["ipAddress"]; ok {
			t.Fatalf("an omitted ip_address must not be sent, body=%v", gotBody)
		}
		if _, ok := gotBody["resourceDescription"]; ok {
			t.Fatalf("an omitted resource_description must not be sent, body=%v", gotBody)
		}
	})

	t.Run("a 201 with no static_ip_id is an error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"success":true,"message":"ok"}`))
		})
		if _, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"}); err == nil {
			t.Fatal("a 201 with an empty static_ip_id must be an error")
		}
	})

	t.Run("an empty static_ip_id is an error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"static_ip_id":""}`))
		})
		if _, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"}); err == nil {
			t.Fatal("an empty static_ip_id must be an error")
		}
	})

	t.Run("a non-201 status is an error", func(t *testing.T) {
		for _, code := range []int{http.StatusOK, http.StatusBadRequest, http.StatusForbidden, http.StatusInternalServerError} {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`{"static_ip_id":"si-x"}`))
			})
			if _, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"}); err == nil {
				t.Fatalf("status %d must be rejected (only 201 is a successful create)", code)
			}
		}
	})
}

// TestVPCStaticIPUpdate pins the ASYNCHRONOUS update contract: a PATCH that
// returns an activity (Location), with a body carrying ONLY the changed
// updatable fields. The body must NEVER contain ipAddress — that pins the
// mutability decision against the UpdateStaticIpPayload swagger schema.
func TestVPCStaticIPUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("PATCH hits the right path, returns the activity id, and forbids ipAddress in the body", func(t *testing.T) {
		var gotBody map[string]any
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch || r.URL.Path != "/vpc/v1/static_ips/si-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			w.Header().Set("Location", "act-99")
			w.WriteHeader(http.StatusCreated)
		})

		desc := "updated"
		mac := "00:50:56:ab:cd:f0"
		activityID, err := c.VPC().StaticIP().Update(ctx, "si-1", &UpdateStaticIPRequest{
			ResourceDescription: &desc,
			MacAddress:          &mac,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if activityID != "act-99" {
			t.Fatalf("the async update must return the activity id from Location, got %q", activityID)
		}
		if gotBody["resourceDescription"] != "updated" || gotBody["macAddress"] != "00:50:56:ab:cd:f0" {
			t.Fatalf("unexpected PATCH body: %v", gotBody)
		}
		// The PATCH payload MUST NOT carry ipAddress (not in UpdateStaticIpPayload).
		if _, ok := gotBody["ipAddress"]; ok {
			t.Fatalf("the PATCH body must never contain ipAddress (it is not updatable), body=%v", gotBody)
		}
		// Exactly the two updatable keys, nothing else.
		if len(gotBody) != 2 {
			t.Fatalf("the PATCH body must contain exactly the changed updatable keys, got %v", gotBody)
		}
	})

	t.Run("a diff-driven PATCH with only resource_description omits macAddress", func(t *testing.T) {
		var gotBody map[string]any
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			w.Header().Set("Location", "act-1")
			w.WriteHeader(http.StatusCreated)
		})
		desc := "only-desc"
		if _, err := c.VPC().StaticIP().Update(ctx, "si-1", &UpdateStaticIPRequest{ResourceDescription: &desc}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := gotBody["macAddress"]; ok {
			t.Fatalf("an unchanged macAddress must be omitted from the PATCH body, got %v", gotBody)
		}
		if gotBody["resourceDescription"] != "only-desc" || len(gotBody) != 1 {
			t.Fatalf("unexpected PATCH body: %v", gotBody)
		}
	})

	t.Run("a PATCH with no Location is an error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated) // no Location header
		})
		desc := "x"
		if _, err := c.VPC().StaticIP().Update(ctx, "si-1", &UpdateStaticIPRequest{ResourceDescription: &desc}); err == nil {
			t.Fatal("an async update with no Location header must be an error")
		}
	})
}

// TestVPCStaticIPDelete pins the ASYNCHRONOUS delete contract: a DELETE that
// returns an activity (Location). A 404 surfaces as a StatusError{404} so the
// resource layer can treat it as an idempotent success.
func TestVPCStaticIPDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("DELETE hits the right path and returns the activity id", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/vpc/v1/static_ips/si-7" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Location", "act-del")
			w.WriteHeader(http.StatusCreated)
		})
		activityID, err := c.VPC().StaticIP().Delete(ctx, "si-7")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if activityID != "act-del" {
			t.Fatalf("delete must return the activity id from Location, got %q", activityID)
		}
	})

	t.Run("a 404 surfaces as a StatusError{404} for idempotent handling", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		_, err := c.VPC().StaticIP().Delete(ctx, "si-gone")
		if err == nil {
			t.Fatal("a 404 must surface as an error so the resource can detect it")
		}
		var statusErr StatusError
		if !errors.As(err, &statusErr) || statusErr.Code != http.StatusNotFound {
			t.Fatalf("a 404 must be a StatusError{404}, got %v", err)
		}
	})
}

// TestVPCStaticIPListStrict pins the deletion-evidence channel: ONLY a complete
// HTTP 200 is usable evidence. A 206 is partial and cannot prove absence;
// 201/403/5xx are not evidence; a malformed 200 is an error.
func TestVPCStaticIPListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 hits the per-private-network path and returns the parsed list", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/vpc/v1/private_networks/pn-1/static_ips" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			// ListStrict must NOT scope the listing to a VM: it must see every IP.
			if r.URL.Query().Get("virtualMachineId") != "" {
				t.Errorf("ListStrict must not send a virtualMachineId filter, got %q", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"si-1","macAddress":"00:50:56:ab:cd:ef"},{"id":"si-2","macAddress":"00:50:56:ab:cd:f0"}]`))
		})
		list, err := c.VPC().StaticIP().ListStrict(ctx, "pn-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 || list[0].ID != "si-1" || list[1].ID != "si-2" {
			t.Fatalf("unexpected list: %+v", list)
		}
	})

	t.Run("200 with malformed JSON errors", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		})
		if _, err := c.VPC().StaticIP().ListStrict(ctx, "pn-1"); err == nil {
			t.Fatal("a malformed 200 body must return a decode error")
		}
	})

	for _, code := range []int{
		http.StatusCreated,             // 201
		http.StatusPartialContent,      // 206 — partial, cannot prove absence
		http.StatusForbidden,           // 403
		http.StatusInternalServerError, // 500
	} {
		code := code
		t.Run(http.StatusText(code)+" is rejected as non-200 evidence", func(t *testing.T) {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`[]`))
			})
			if _, err := c.VPC().StaticIP().ListStrict(ctx, "pn-1"); err == nil {
				t.Fatalf("status %d must FAIL CLOSED: only a 200 is deletion evidence", code)
			}
		})
	}
}
