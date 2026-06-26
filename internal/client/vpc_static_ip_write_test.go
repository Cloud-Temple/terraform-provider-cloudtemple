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

	// The LIVE API returns 201 with an EMPTY body (the swagger's static_ip_id is
	// not actually sent). The create is synchronous, so the id MUST be resolved
	// from a complete listing of the private network rather than treated as a
	// failure — a failure would orphan the created static IP (created
	// platform-side but absent from the Terraform state). The resolution matches
	// the requested MAC AND source=="custom".
	//
	// Non-complacent on TWO axes: (1) the pre-fix code returned an error on an
	// empty body, so an empty-body success goes RED without the fallback; (2) an
	// "xoa" static IP shares the SAME MAC (the #311 platform auto-creation), and
	// it is listed FIRST — without the source=="custom" filter the resolution
	// would match both and either pick xoa or fail ambiguous, never returning the
	// custom id this assertion demands.
	t.Run("a 201 empty body resolves the id from the listing, picking the custom IP over a co-resident xoa one", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				w.WriteHeader(http.StatusCreated) // empty body, like the live API
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/private_networks/pn-1/static_ips":
				w.WriteHeader(http.StatusOK)
				// xoa first (same MAC, #311 co-residence), then our custom one.
				_, _ = w.Write([]byte(`[
					{"id":"si-xoa","macAddress":"00:50:56:AB:CD:EF","source":"xoa"},
					{"id":"si-custom","macAddress":"00:50:56:ab:cd:ef","source":"custom"},
					{"id":"si-other","macAddress":"00:50:56:11:22:33","source":"custom"}
				]`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		id, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"})
		if err != nil {
			t.Fatalf("an empty 201 body must resolve the id from the listing, not fail: %v", err)
		}
		if id != "si-custom" {
			t.Fatalf("the id must resolve to the CUSTOM static IP for the MAC, got %q (a %q here means the xoa co-resident IP was wrongly picked)", id, "si-xoa")
		}
	})

	// The API may format the returned MAC differently from the request (the
	// swagger pattern tolerates "-" and any case). Resolution must normalise both
	// sides, or a well-created static IP returned as "00-50-56-AB-CD-EF" would not
	// match the requested "00:50:56:ab:cd:ef" and would be orphaned.
	// Non-complacent: a raw == comparison reds this case.
	t.Run("a 201 empty body matches the custom IP across MAC case/separator differences", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated) // empty body
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"si-fmt","macAddress":"00-50-56-AB-CD-EF","source":"custom"}]`))
		})
		id, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"})
		if err != nil {
			t.Fatalf("MAC formatting differences must be normalised, not fail: %v", err)
		}
		if id != "si-fmt" {
			t.Fatalf("the id must resolve across MAC formatting, got %q", id)
		}
	})

	t.Run("a 201 empty body with no matching custom IP in the listing is an error (never a silent success)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated) // empty body
				return
			}
			w.WriteHeader(http.StatusOK)
			// Only a co-resident xoa IP for the MAC, plus an unrelated custom one.
			_, _ = w.Write([]byte(`[
				{"id":"si-xoa","macAddress":"00:50:56:ab:cd:ef","source":"xoa"},
				{"id":"si-other","macAddress":"00:50:56:11:22:33","source":"custom"}
			]`))
		})
		if _, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"}); err == nil {
			t.Fatal("no matching custom IP must error, not silently succeed (or bind to the xoa one)")
		}
	})

	t.Run("a 201 empty body with a failing strict listing is an error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated) // empty body
				return
			}
			// A 206/403/5xx is rejected by ListStrict -> cannot resolve -> error.
			w.WriteHeader(http.StatusForbidden)
		})
		if _, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"}); err == nil {
			t.Fatal("an empty body whose listing fails must error, not silently succeed")
		}
	})

	// A listing entry WITHOUT an id must error rather than resolve to an empty id
	// (which would orphan the created static IP via SetId("")). This is now caught
	// upstream by ListStrict's structural guard (an id-less entry makes the whole
	// listing untrusted); Create keeps a defensive match.ID=="" check too.
	t.Run("a 201 empty body whose listing has an id-less entry is an error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated) // empty body
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"macAddress":"00:50:56:ab:cd:ef","source":"custom"}]`)) // no id
		})
		if _, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"}); err == nil {
			t.Fatal("a matched custom entry without an id must error, not return an empty id with nil error")
		}
	})

	t.Run("a 201 empty body with two custom IPs sharing the MAC is an ambiguity error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated) // empty body
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id":"si-a","macAddress":"00:50:56:ab:cd:ef","source":"custom"},
				{"id":"si-b","macAddress":"00:50:56:ab:cd:ef","source":"custom"}
			]`))
		})
		if _, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef"}); err == nil {
			t.Fatal("two custom IPs sharing the MAC must be an ambiguity error, not a silent pick")
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

	// A 200 "null" body decodes to an empty slice with json.Decoder — it would be
	// a FALSE "empty network" and could drop state. It must fail closed.
	// Non-complacent: the old decodeBody path accepted it as an empty listing.
	t.Run("200 with a null body is rejected (not a provable empty listing)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`null`))
		})
		if _, err := c.VPC().StaticIP().ListStrict(ctx, "pn-1"); err == nil {
			t.Fatal("a 200 null body must be rejected, never read as an empty (absent) network")
		}
	})

	t.Run("200 with a JSON object (not an array) is rejected", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"static_ips":[]}`))
		})
		if _, err := c.VPC().StaticIP().ListStrict(ctx, "pn-1"); err == nil {
			t.Fatal("a 200 object body must be rejected: only a JSON array can prove completeness")
		}
	})

	// An entry without an id makes id-matching unreliable: our static IP could be
	// the malformed entry and be wrongly judged absent. Fail closed.
	// Non-complacent: without the per-entry id check this listing is accepted.
	t.Run("200 with an entry missing its id is rejected (structurally incomplete)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"si-1"},{"macAddress":"00:50:56:ab:cd:ff"}]`))
		})
		if _, err := c.VPC().StaticIP().ListStrict(ctx, "pn-1"); err == nil {
			t.Fatal("a listing with an id-less entry must be rejected as structurally incomplete")
		}
	})

	t.Run("200 with an empty JSON array is a valid (genuinely empty) listing", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		})
		list, err := c.VPC().StaticIP().ListStrict(ctx, "pn-1")
		if err != nil {
			t.Fatalf("an empty JSON array is a provably empty listing, must not error: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected an empty list, got %d entries", len(list))
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
