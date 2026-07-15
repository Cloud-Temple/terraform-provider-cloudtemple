package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// completedActivityBody returns the minimal /activity/v1/activities/{id} payload
// WaitForCompletion polls and treats as terminal success. The Activity struct
// carries no json tags, so encoding/json matches "state"/"result"
// case-insensitively; the map key MUST be "completed" (the only state
// WaitForCompletion accepts as done) and Result carries the created static IP id.
// (Named *Body to avoid colliding with activity_unit_test.go's completedActivity,
// which builds a *Activity struct for the polling-loop unit tests.)
func completedActivityBody(result string) string {
	return fmt.Sprintf(`{"id":"act-create","state":{"completed":{"result":%q}}}`, result)
}

// TestVPCStaticIPCreateStart pins the create POST contract WITHOUT waiting. It
// reports EXACTLY ONE of activityID (the live ASYNC path: 201 + Location, empty
// body) or syncID (the defensive SYNC path: 201 + body static_ip_id), posts the
// right path and body (resourceDescription ALWAYS present — no omitempty), and
// rejects a doomed request (empty/whitespace resourceDescription) BEFORE the POST.
func TestVPCStaticIPCreateStart(t *testing.T) {
	ctx := context.Background()

	t.Run("async 201+Location returns ONLY the activity id, posting the right path/body", func(t *testing.T) {
		var gotBody map[string]any
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vpc/v1/private_networks/pn-1/static_ips" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			w.Header().Set("Location", "act-create")
			w.WriteHeader(http.StatusCreated) // empty body, like the live API
		})

		activityID, syncID, err := c.VPC().StaticIP().CreateStart(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			IPAddress:           "10.0.1.50",
			ResourceDescription: "web",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// XOR: a successful async create yields the activity id and NOTHING in syncID.
		if activityID != "act-create" || syncID != "" {
			t.Fatalf("async create must return ONLY the activity id, got activityID=%q syncID=%q", activityID, syncID)
		}
		if gotBody["macAddress"] != "00:50:56:ab:cd:ef" || gotBody["ipAddress"] != "10.0.1.50" || gotBody["resourceDescription"] != "web" {
			t.Fatalf("unexpected create body: %v", gotBody)
		}
	})

	t.Run("sync 201+body static_ip_id (no Location) returns ONLY the body id", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			// No Location: the defensive sync path resolves the id from the body.
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"static_ip_id":"si-sync"}`))
		})
		activityID, syncID, err := c.VPC().StaticIP().CreateStart(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			ResourceDescription: "web",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// XOR: a successful sync create yields the body id and NOTHING in activityID.
		if syncID != "si-sync" || activityID != "" {
			t.Fatalf("sync create must return ONLY the body id, got activityID=%q syncID=%q", activityID, syncID)
		}
	})

	t.Run("a 201 with neither a Location nor a static_ip_id body fails closed", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated) // empty body, no Location
		})
		if _, _, err := c.VPC().StaticIP().CreateStart(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			ResourceDescription: "web",
		}); err == nil {
			t.Fatal("a 201 carrying no id signal must fail closed (the pre-create teardown is the orphan net), never guess an id by listing")
		}
	})

	t.Run("an omitted ipAddress is not sent; a provided resourceDescription is forwarded as-is", func(t *testing.T) {
		var gotBody map[string]any
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			w.Header().Set("Location", "act-create")
			w.WriteHeader(http.StatusCreated)
		})
		if _, _, err := c.VPC().StaticIP().CreateStart(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			ResourceDescription: "web",
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// ipAddress keeps omitempty: an omitted (zero) value must NOT be in the body.
		// Mutation-proof — drop ipAddress's omitempty and "" would serialise, failing here.
		if _, ok := gotBody["ipAddress"]; ok {
			t.Fatalf("an omitted ipAddress must not be sent, body=%v", gotBody)
		}
		// A provided resourceDescription is forwarded with its value intact. This does
		// NOT by itself pin the no-omitempty tag (a non-empty value serialises either
		// way); the tag is pinned directly by the marshal subtest below, and the
		// REQUIRED-reject-empty contract by the precondition subtests further down.
		if gotBody["resourceDescription"] != "web" {
			t.Fatalf("resourceDescription must be forwarded as-is, body=%v", gotBody)
		}
	})

	t.Run("resourceDescription carries no omitempty: an empty value is still serialised", func(t *testing.T) {
		// The struct deliberately drops omitempty on resourceDescription (the live
		// contract REQUIRES the field). Pin that directly on the type's serialisation,
		// independently of CreateStart's precondition. Mutation-proof: re-add omitempty
		// and the empty value is elided, so the field vanishes and this assertion fails.
		b, err := json.Marshal(&CreateStaticIPRequest{MacAddress: "00:50:56:ab:cd:ef", ResourceDescription: ""})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(b), `"resourceDescription":""`) {
			t.Fatalf("an empty resourceDescription must still be serialised (no omitempty); got %s", b)
		}
		// Conversely, the omitempty field IS elided when empty (same marshal, opposite
		// contract) — so this subtest also guards against accidentally dropping
		// ipAddress's omitempty.
		if strings.Contains(string(b), "ipAddress") {
			t.Fatalf("an empty ipAddress must be elided (omitempty); got %s", b)
		}
	})

	// resourceDescription is REQUIRED by the live create contract: an empty or
	// whitespace-only value is rejected as a CreateStart PRECONDITION error BEFORE
	// any POST, so a doomed request never reaches the API. Non-complacent: a
	// `posted` flag asserts the rejection happens BEFORE the HTTP call — not that
	// the server happened to bounce it. Replaces the old "omitted resource_
	// description is silently not sent" test, which encoded the wrong contract.
	for _, tc := range []struct {
		name string
		desc string
	}{
		{"empty resourceDescription", ""},
		{"whitespace-only resourceDescription", "   "},
	} {
		tc := tc
		t.Run(tc.name+" is rejected before the POST", func(t *testing.T) {
			var posted bool
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					posted = true
				}
				w.WriteHeader(http.StatusCreated)
			})
			_, _, err := c.VPC().StaticIP().CreateStart(ctx, "pn-1", &CreateStaticIPRequest{
				MacAddress:          "00:50:56:ab:cd:ef",
				ResourceDescription: tc.desc,
			})
			if err == nil {
				t.Fatal("an empty/whitespace resourceDescription must be a precondition error")
			}
			if posted {
				t.Fatal("the precondition must reject BEFORE the POST (no doomed request reaches the API)")
			}
		})
	}

	t.Run("a non-201 status is an error", func(t *testing.T) {
		for _, code := range []int{http.StatusOK, http.StatusBadRequest, http.StatusForbidden, http.StatusInternalServerError} {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`{"static_ip_id":"si-x"}`))
			})
			if _, _, err := c.VPC().StaticIP().CreateStart(ctx, "pn-1", &CreateStaticIPRequest{
				MacAddress:          "00:50:56:ab:cd:ef",
				ResourceDescription: "web",
			}); err == nil {
				t.Fatalf("status %d must be rejected (only 201 is a successful create)", code)
			}
		}
	})

	t.Run("an EXPECTED create activity is not rejected by ErrorOnUnexpectedActivity", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "act-create")
			w.WriteHeader(http.StatusCreated)
		})
		// internal/client/tests sets this guard suite-wide to catch sync endpoints that
		// unexpectedly go async. CreateStart EXPECTS an activity, so — like the sibling
		// async methods Update/Delete — it must NOT trip the guard. Mutation-proof: route
		// CreateStart back through doRequest/doRequestOnce and this goes red with
		// "an unexpected Location header has been found".
		c.config.ErrorOnUnexpectedActivity = true
		activityID, syncID, err := c.VPC().StaticIP().CreateStart(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			ResourceDescription: "web",
		})
		if err != nil {
			t.Fatalf("CreateStart expects an activity and must bypass ErrorOnUnexpectedActivity; got error: %v", err)
		}
		if activityID != "act-create" || syncID != "" {
			t.Fatalf("expected activityID=act-create syncID=%q; got activityID=%q syncID=%q", "", activityID, syncID)
		}
	})
}

// TestVPCStaticIPWaitCreate pins the activity-resolution contract: the new id is
// the completed activity's single state Result; an EMPTY Result fails closed
// (R-M1) — a created id we cannot read must be an error, never an empty id that
// would orphan the resource via SetId(""). options is nil here (the WaiterOptions
// methods are nil-safe), since these tests pin resolution, not poll logging.
func TestVPCStaticIPWaitCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("a completed activity yields the id from its single state Result", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/activity/v1/activities/act-create" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(completedActivityBody("si-new")))
		})
		id, err := c.VPC().StaticIP().WaitCreate(ctx, "act-create", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "si-new" {
			t.Fatalf("the id must come from the activity Result, got %q", id)
		}
	})

	t.Run("a completed activity with an EMPTY Result fails closed (R-M1)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(completedActivityBody("")))
		})
		if _, err := c.VPC().StaticIP().WaitCreate(ctx, "act-create", nil); err == nil {
			t.Fatal("an empty activity Result must fail closed, never return an empty id with a nil error")
		}
	})
}

// TestVPCStaticIPCreate pins the composed async create (CreateStart + WaitCreate):
// it POSTs the create, waits on the Location activity, resolves the id from that
// activity's Result, and — crucially — performs NO listing GET (the old
// orphan-prone MAC-resolution path is gone, R-M2). A wait failure yields an error
// carrying the activityID (R-Q2: the provider must fail closed and not SetId), and
// a sync body id short-circuits the wait entirely.
func TestVPCStaticIPCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("async happy path resolves the id from the activity and never lists the network", func(t *testing.T) {
		var sawListing bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/vpc/v1/private_networks/pn-1/static_ips":
				w.Header().Set("Location", "act-create")
				w.WriteHeader(http.StatusCreated) // empty body, live contract
			case r.Method == http.MethodGet && r.URL.Path == "/activity/v1/activities/act-create":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(completedActivityBody("si-new")))
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/private_networks/pn-1/static_ips":
				// The async create must NEVER resolve the id by listing the network.
				sawListing = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[]`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		id, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			ResourceDescription: "web",
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "si-new" {
			t.Fatalf("id must resolve from the activity Result, got %q", id)
		}
		if sawListing {
			t.Fatal("the async create happy path must NOT list the network (the orphan-prone resolution path is removed)")
		}
	})

	t.Run("a wait failure yields an error carrying the activityID, never a usable id", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				w.Header().Set("Location", "act-create")
				w.WriteHeader(http.StatusCreated)
			case r.Method == http.MethodGet && r.URL.Path == "/activity/v1/activities/act-create":
				// Completed but EMPTY Result → WaitCreate fails closed (R-M1).
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(completedActivityBody("")))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})
		id, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			ResourceDescription: "web",
		}, nil)
		if err == nil {
			t.Fatal("a wait failure must surface as an error, never a usable id")
		}
		if id != "" {
			t.Fatalf("a failed create must return an empty id, got %q", id)
		}
		// The wrap must carry the activityID so a postmortem can name the orphan window.
		if !strings.Contains(err.Error(), "act-create") {
			t.Fatalf("the wait-failure error must carry the activityID for correlation, got %v", err)
		}
	})

	t.Run("the sync path returns the body id directly without polling or listing", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("the sync path must POST only (no activity poll, no listing), got %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"static_ip_id":"si-sync"}`))
		})
		id, err := c.VPC().StaticIP().Create(ctx, "pn-1", &CreateStaticIPRequest{
			MacAddress:          "00:50:56:ab:cd:ef",
			ResourceDescription: "web",
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "si-sync" {
			t.Fatalf("the sync path must return the body id, got %q", id)
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
