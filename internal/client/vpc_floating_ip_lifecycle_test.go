package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// This file pins the C4 floating-IP LIFECYCLE contract (provision / describe /
// deprovision / strict by-id resolve), distinct from vpc_floating_ip_write_test.go
// which pins the bind/unbind/corroborate (C5) surface. A floating IP is a BILLABLE
// public IP with NO client idempotency key and a count-only provision body, so the
// state-safety doctrine is at full strength here: every destructive decision rests
// on strict positive evidence and fails closed otherwise.

// failedActivityBody is the minimal /activity payload WaitForCompletion treats as a
// TERMINAL FAILURE (the "failed" state). It is the counterpart to
// completedActivityBody (vpc_static_ip_write_test.go) and lets the lifecycle tests
// drive the WaitForCompletion error path without HTTP transport errors.
func failedActivityBody() string {
	return `{"id":"act-prov","state":{"failed":{}}}`
}

// TestVPCFloatingIPProvisionStart pins the provision POST contract WITHOUT waiting.
// It posts the EXACT count-only body ({"count":1} — no vpcId, no description,
// confirmed by live probe), reports EXACTLY ONE of activityID (live async: 201 +
// Location, empty body) or syncID (defensive sync: 201 + body id), and fails closed
// on a 201 that carries neither signal (a billable IP may have been allocated, so
// the never-orphan net is the caller's audit, never a silent id guess).
func TestVPCFloatingIPProvisionStart(t *testing.T) {
	ctx := context.Background()

	t.Run("async 201+Location returns ONLY the activity id, posting count-only body", func(t *testing.T) {
		var gotBody map[string]any
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vpc/v1/floating_ips" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			w.Header().Set("Location", "act-prov")
			w.WriteHeader(http.StatusCreated) // empty body, like the live API
		})

		activityID, syncID, err := c.VPC().FloatingIP().ProvisionStart(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// XOR: a successful async provision yields the activity id and NOTHING else.
		if activityID != "act-prov" || syncID != "" {
			t.Fatalf("async provision must return ONLY the activity id, got activityID=%q syncID=%q", activityID, syncID)
		}
		// The body MUST be exactly {"count":1}: no vpcId, no description, count==1.
		// Mutation-proof — add any field to provisionFloatingIPRequest and len!=1 fails.
		if gotBody["count"] != float64(1) || len(gotBody) != 1 {
			t.Fatalf("provision body must be exactly {\"count\":1}, got %v", gotBody)
		}
	})

	t.Run("sync 201+body id (no Location) returns ONLY the body id", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			// No Location: the defensive sync path resolves the id from the body.
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"fip-sync"}`))
		})
		activityID, syncID, err := c.VPC().FloatingIP().ProvisionStart(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// XOR: a successful sync provision yields the body id and NOTHING in activityID.
		if syncID != "fip-sync" || activityID != "" {
			t.Fatalf("sync provision must return ONLY the body id, got activityID=%q syncID=%q", activityID, syncID)
		}
	})

	t.Run("a 201 with neither a Location nor a body id fails closed", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated) // empty body, no Location
		})
		if _, _, err := c.VPC().FloatingIP().ProvisionStart(ctx); err == nil {
			t.Fatal("a 201 carrying no id signal must fail closed (a billable IP may have been allocated; the audit is the net, never an id guess)")
		}
	})

	t.Run("a non-201 status is an error", func(t *testing.T) {
		for _, code := range []int{http.StatusOK, http.StatusBadRequest, http.StatusForbidden, http.StatusInternalServerError} {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`{"id":"fip-x"}`))
			})
			if _, _, err := c.VPC().FloatingIP().ProvisionStart(ctx); err == nil {
				t.Fatalf("status %d must be rejected (only 201 is a successful provision)", code)
			}
		}
	})

	t.Run("an EXPECTED provision activity is not rejected by ErrorOnUnexpectedActivity", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "act-prov")
			w.WriteHeader(http.StatusCreated)
		})
		// The suite-wide guard catches sync endpoints that unexpectedly go async.
		// ProvisionStart EXPECTS a Location, so — like CreateStart — it must bypass
		// the guard via doRequestWithToken. Mutation-proof: route ProvisionStart back
		// through doRequest and this goes red with "unexpected Location header".
		c.config.ErrorOnUnexpectedActivity = true
		activityID, syncID, err := c.VPC().FloatingIP().ProvisionStart(ctx)
		if err != nil {
			t.Fatalf("ProvisionStart expects an activity and must bypass ErrorOnUnexpectedActivity; got error: %v", err)
		}
		if activityID != "act-prov" || syncID != "" {
			t.Fatalf("expected activityID=act-prov syncID=\"\"; got activityID=%q syncID=%q", activityID, syncID)
		}
	})

	t.Run("the provision body type marshals to exactly {\"count\":1}", func(t *testing.T) {
		// Pin the type's serialisation directly, independently of the POST. The body
		// MUST be count-only: no description, no vpcId. Mutation-proof — add a field
		// or rename the json tag and the exact-match fails.
		b, err := json.Marshal(provisionFloatingIPRequest{Count: 1})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(b) != `{"count":1}` {
			t.Fatalf("provision body must marshal to exactly {\"count\":1}, got %s", b)
		}
	})
}

// TestVPCFloatingIPWaitProvision pins the activity-resolution contract of the
// provision wait. It shares the R-M1 core with static WaitCreate
// (waitCreatedIDFromActivity, vpc.go): the id is the completed activity's single
// state Result, and an EMPTY Result FAILS CLOSED — a provisioned (billable) IP we
// cannot name must surface as an error, never an empty id.
//
// NOTE on coverage: the helper's `act == nil || len(act.State) != 1` guard is
// DEFENSIVE DEPTH restating an invariant enforced UPSTREAM at the source —
// activity.go treats a read with len(State) != 1 as a RETRYABLE error
// (activity.go:283-287), so WaitForCompletion NEVER returns such an activity (it
// retries to the deadline and returns an error). The helper guard is therefore
// structurally unreachable through the real WaitForCompletion path this helper
// calls, so it is NOT faked here: injecting a malformed activity would test an
// impossible path. The reachable branches (id extraction, empty-Result fail-closed,
// WaitForCompletion error propagation) ARE pinned below. NB: that upstream
// len(State)!=1 retry is the load-bearing invariant but has no DEDICATED unit test of
// its own today; this comment does not claim one — it is verifiable at
// activity.go:283-287 (a separate, pre-existing activity.go coverage gap, out of C4 scope).
func TestVPCFloatingIPWaitProvision(t *testing.T) {
	ctx := context.Background()

	t.Run("a completed activity yields the id from its single state Result", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/activity/v1/activities/act-prov" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(completedActivityBody("fip-new")))
		})
		id, err := c.VPC().FloatingIP().WaitProvision(ctx, "act-prov", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "fip-new" {
			t.Fatalf("the id must come from the activity Result, got %q", id)
		}
	})

	t.Run("a completed activity with an EMPTY Result fails closed, labelled 'floating IP provision'", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(completedActivityBody("")))
		})
		_, err := c.VPC().FloatingIP().WaitProvision(ctx, "act-prov", nil)
		if err == nil {
			t.Fatal("an empty activity Result must fail closed, never return an empty id with a nil error")
		}
		// Label parity: the shared helper is parameterised by label, so the FIP path
		// must surface "floating IP provision" (NOT the static "static IP create").
		// Mutation-proof — hardcode the static label in the helper and this fails.
		if !strings.Contains(err.Error(), "floating IP provision") {
			t.Fatalf("the FIP wait error must carry the 'floating IP provision' label, got %v", err)
		}
	})

	t.Run("a terminal failed activity surfaces the WaitForCompletion error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(failedActivityBody()))
		})
		if _, err := c.VPC().FloatingIP().WaitProvision(ctx, "act-prov", nil); err == nil {
			t.Fatal("a failed provision activity must surface as an error, never a usable id")
		}
	})
}

// TestVPCFloatingIPProvision pins the composed provision (ProvisionStart +
// WaitProvision): the SYNC body id short-circuits and is returned WITHOUT ever
// waiting (never WaitProvision("")), and a wait failure is wrapped WITH the
// activityID and the billable-audit recovery text and NEVER yields (id, nil).
func TestVPCFloatingIPProvision(t *testing.T) {
	ctx := context.Background()

	t.Run("async happy path resolves the id from the provision activity", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/vpc/v1/floating_ips":
				w.Header().Set("Location", "act-prov")
				w.WriteHeader(http.StatusCreated)
			case r.Method == http.MethodGet && r.URL.Path == "/activity/v1/activities/act-prov":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(completedActivityBody("fip-new")))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		id, err := c.VPC().FloatingIP().Provision(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "fip-new" {
			t.Fatalf("id must resolve from the activity Result, got %q", id)
		}
	})

	t.Run("the sync path returns the body id directly WITHOUT polling an activity", func(t *testing.T) {
		var sawActivity bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/vpc/v1/floating_ips":
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"fip-sync"}`))
			case strings.HasPrefix(r.URL.Path, "/activity/"):
				// A sync provision MUST short-circuit: never WaitProvision("").
				sawActivity = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(completedActivityBody("WRONG")))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})
		id, err := c.VPC().FloatingIP().Provision(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "fip-sync" {
			t.Fatalf("the sync path must return the body id, got %q", id)
		}
		if sawActivity {
			t.Fatal("the sync path must NOT poll an activity (never WaitProvision with an empty activity id)")
		}
	})

	t.Run("a wait failure yields an error carrying the activityID and billable-audit text, never a usable id", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				w.Header().Set("Location", "act-prov")
				w.WriteHeader(http.StatusCreated)
			case r.Method == http.MethodGet && r.URL.Path == "/activity/v1/activities/act-prov":
				// Completed but EMPTY Result → WaitProvision fails closed (R-M1).
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(completedActivityBody("")))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		})
		id, err := c.VPC().FloatingIP().Provision(ctx, nil)
		if err == nil {
			t.Fatal("a wait failure must surface as an error, never a usable id")
		}
		if id != "" {
			t.Fatalf("a failed provision must return an empty id, got %q", id)
		}
		// The wrap must name the activity (orphan correlation) AND point at the
		// billable-audit recovery — this is a BILLABLE IP, so the operator must know
		// to go look for it.
		if !strings.Contains(err.Error(), "act-prov") {
			t.Fatalf("the wait-failure error must carry the activityID, got %v", err)
		}
		if !strings.Contains(err.Error(), "audit") {
			t.Fatalf("the wait-failure error must point at the billable-audit recovery, got %v", err)
		}
	})
}

// TestVPCFloatingIPResolveByID pins the STRICT tri-state by-id read. For a floating
// IP there is NO listing-omission drop channel (unlike static IP), so a by-id 404
// is the SOLE absence signal: 200 → id-consistency-guarded present; 404 →
// authoritative absent; 403 → keep+fail (forbidden is not absence, #303);
// 206/other → keep+fail. A 200 whose body id is empty or mismatched fails closed.
func TestVPCFloatingIPResolveByID(t *testing.T) {
	ctx := context.Background()

	t.Run("200 with a matching id hits the by-id path and returns present", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/vpc/v1/floating_ips/fip-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fip-1","ipAddress":"198.51.100.1","description":"web","staticIp":null}`))
		})
		fip, found, err := c.VPC().FloatingIP().ResolveByID(ctx, "fip-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !found || fip == nil || fip.ID != "fip-1" || fip.Description != "web" {
			t.Fatalf("a 200 with a matching id must be present, got found=%v fip=%+v", found, fip)
		}
	})

	t.Run("200 whose body id does NOT match the requested id fails closed", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fip-OTHER"}`))
		})
		if _, _, err := c.VPC().FloatingIP().ResolveByID(ctx, "fip-1"); err == nil {
			t.Fatal("a 200 whose body id mismatches the requested id must fail closed (never trust a mismatched body)")
		}
	})

	t.Run("200 with an empty body id fails closed", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ipAddress":"198.51.100.1"}`))
		})
		if _, _, err := c.VPC().FloatingIP().ResolveByID(ctx, "fip-1"); err == nil {
			t.Fatal("a 200 with an empty id must fail closed (an id-less body cannot represent the requested FIP)")
		}
	})

	t.Run("404 is the AUTHORITATIVE absent (the sole drop channel)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		fip, found, err := c.VPC().FloatingIP().ResolveByID(ctx, "fip-gone")
		if err != nil {
			t.Fatalf("a 404 must be a clean (nil err) absent, got %v", err)
		}
		if found || fip != nil {
			t.Fatalf("a 404 must be absent (found=false, nil fip), got found=%v fip=%+v", found, fip)
		}
	})

	t.Run("403 is NOT absence: keep + fail (#303)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		_, found, err := c.VPC().FloatingIP().ResolveByID(ctx, "fip-1")
		if err == nil {
			t.Fatal("a 403 must surface as an error, never be collapsed into absent")
		}
		if found {
			t.Fatal("a 403 must NOT report found")
		}
	})

	t.Run("206 and other non-200/404/403 statuses keep + fail", func(t *testing.T) {
		for _, code := range []int{http.StatusPartialContent, http.StatusInternalServerError, http.StatusBadGateway} {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			})
			if _, found, err := c.VPC().FloatingIP().ResolveByID(ctx, "fip-1"); err == nil || found {
				t.Fatalf("status %d must keep+fail (never absence), got found=%v err=%v", code, found, err)
			}
		}
	})
}

// TestVPCFloatingIPUpdateDescription pins the conditional-wait PATCH. The
// floating-IP PATCH may be sync OR async, so the status is classified FIRST and a
// Location is read as an async handle ONLY on a success status. The crucial
// fail-closed is the 202-without-Location: a 202 Accepted with no activity handle
// is async-without-a-handle and CANNOT be confirmed, so it must never report success.
func TestVPCFloatingIPUpdateDescription(t *testing.T) {
	ctx := context.Background()

	t.Run("PATCH posts {\"description\":value} to the by-id path and returns the activity id on success+Location", func(t *testing.T) {
		var gotBody map[string]any
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch || r.URL.Path != "/vpc/v1/floating_ips/fip-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			w.Header().Set("Location", "act-desc")
			w.WriteHeader(http.StatusOK)
		})
		activityID, err := c.VPC().FloatingIP().UpdateDescription(ctx, "fip-1", "managed-desc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if activityID != "act-desc" {
			t.Fatalf("a success+Location PATCH must return the activity id, got %q", activityID)
		}
		// The body MUST be exactly {"description":"managed-desc"}.
		if gotBody["description"] != "managed-desc" || len(gotBody) != 1 {
			t.Fatalf("PATCH body must be exactly {\"description\":...}, got %v", gotBody)
		}
	})

	t.Run("200 and 204 WITHOUT a Location are sync success (no wait, empty activity id)", func(t *testing.T) {
		for _, code := range []int{http.StatusOK, http.StatusNoContent} {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code) // no Location
			})
			activityID, err := c.VPC().FloatingIP().UpdateDescription(ctx, "fip-1", "d")
			if err != nil {
				t.Fatalf("status %d without a Location must be a sync success, got error: %v", code, err)
			}
			if activityID != "" {
				t.Fatalf("a sync PATCH has no activity to wait on; the id must be empty, got %q", activityID)
			}
		}
	})

	t.Run("202 WITHOUT a Location fails closed (async-without-a-handle, unconfirmable)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted) // 202, no Location
		})
		if _, err := c.VPC().FloatingIP().UpdateDescription(ctx, "fip-1", "d"); err == nil {
			t.Fatal("a 202 Accepted with no Location must FAIL CLOSED: the update cannot be confirmed, so it is never reported as success")
		}
	})

	t.Run("202 WITH a Location is an async success returning the activity id", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "act-desc")
			w.WriteHeader(http.StatusAccepted)
		})
		activityID, err := c.VPC().FloatingIP().UpdateDescription(ctx, "fip-1", "d")
		if err != nil {
			t.Fatalf("a 202 WITH a Location is async success, got error: %v", err)
		}
		if activityID != "act-desc" {
			t.Fatalf("a 202+Location must return the activity id, got %q", activityID)
		}
	})

	t.Run("206 and 400 are errors (only {200,201,202,204} are accepted)", func(t *testing.T) {
		for _, code := range []int{http.StatusPartialContent, http.StatusBadRequest} {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			})
			if _, err := c.VPC().FloatingIP().UpdateDescription(ctx, "fip-1", "d"); err == nil {
				t.Fatalf("status %d must be rejected by requireHttpCodes", code)
			}
		}
	})

	t.Run("an EXPECTED async PATCH is not rejected by ErrorOnUnexpectedActivity", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "act-desc")
			w.WriteHeader(http.StatusOK)
		})
		// Like CreateStart/ProvisionStart, UpdateDescription uses doRequestWithToken so
		// a legitimate Location bypasses the suite-wide guard. Mutation-proof: route it
		// through doRequest and this goes red on the EXPECTED Location.
		c.config.ErrorOnUnexpectedActivity = true
		if _, err := c.VPC().FloatingIP().UpdateDescription(ctx, "fip-1", "d"); err != nil {
			t.Fatalf("an async PATCH must bypass ErrorOnUnexpectedActivity, got error: %v", err)
		}
	})

	t.Run("the description body type marshals an empty value (no omitempty)", func(t *testing.T) {
		// The desired description is always non-empty (schema Default +
		// StringIsNotWhiteSpace), but the field must carry NO omitempty so the PATCH
		// always sends it. Mutation-proof — add omitempty and the empty value is
		// elided, so the field vanishes and this assertion fails.
		b, err := json.Marshal(updateFloatingIPDescriptionRequest{Description: ""})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(b) != `{"description":""}` {
			t.Fatalf("an empty description must still be serialised (no omitempty), got %s", b)
		}
	})
}

// TestVPCFloatingIPDeprovisionRaw pins the UNEXPORTED raw DELETE outcome
// classification (no gating, no confirm — those live in DeprovisionUnbound). The
// status is classified FIRST and a Location is read ONLY on success: 404 → gone;
// success+Location → activityID; success-no-Location → sync; 403 → error (forbidden
// is NOT deleted, #303); 206/other → error (never "gone").
func TestVPCFloatingIPDeprovisionRaw(t *testing.T) {
	ctx := context.Background()

	t.Run("404 is idempotent gone", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/vpc/v1/floating_ips/fip-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusNotFound)
		})
		activityID, gone, err := c.VPC().FloatingIP().deprovisionRaw(ctx, "fip-1")
		if err != nil || !gone || activityID != "" {
			t.Fatalf("a 404 must be ('', true, nil), got activityID=%q gone=%v err=%v", activityID, gone, err)
		}
	})

	t.Run("success codes WITH a Location return the activity id (not gone)", func(t *testing.T) {
		for _, code := range []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusCreated} {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "act-del")
				w.WriteHeader(code)
			})
			activityID, gone, err := c.VPC().FloatingIP().deprovisionRaw(ctx, "fip-1")
			if err != nil || gone || activityID != "act-del" {
				t.Fatalf("status %d WITH a Location must be (act-del, false, nil), got activityID=%q gone=%v err=%v", code, activityID, gone, err)
			}
		}
	})

	t.Run("a success code WITHOUT a Location is a sync deprovision (not gone, no activity)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent) // no Location
		})
		activityID, gone, err := c.VPC().FloatingIP().deprovisionRaw(ctx, "fip-1")
		if err != nil || gone || activityID != "" {
			t.Fatalf("a sync 204 must be ('', false, nil), got activityID=%q gone=%v err=%v", activityID, gone, err)
		}
	})

	t.Run("403 is an error, never 'gone' (#303)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		_, gone, err := c.VPC().FloatingIP().deprovisionRaw(ctx, "fip-1")
		if err == nil || gone {
			t.Fatalf("a 403 must be an error and NOT gone, got gone=%v err=%v", gone, err)
		}
	})

	t.Run("206 and 500 are errors, never 'gone'", func(t *testing.T) {
		for _, code := range []int{http.StatusPartialContent, http.StatusInternalServerError} {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			})
			if _, gone, err := c.VPC().FloatingIP().deprovisionRaw(ctx, "fip-1"); err == nil || gone {
				t.Fatalf("status %d must be an error and NOT gone, got gone=%v err=%v", code, gone, err)
			}
		}
	})
}

// TestVPCFloatingIPBodyProvesFullyUnbound pins the deprovision GATE predicate
// directly: it keys on the RAW body, not the decoded struct, because an OMITTED
// association field decodes to a nil pointer IDENTICALLY to an explicit null. The
// destructive path must therefore demand POSITIVE present-and-null evidence:
//
//   - all three associations PRESENT and explicitly null -> true (the only proof).
//   - ANY association field OMITTED -> false. This is the core never-delete-on-weak-
//     evidence invariant: a partial/projection body that drops the fields must NOT
//     read as "fully unbound" (it would otherwise green-light deleting a billable,
//     possibly-bound IP). `{"id":"fip-1"}` alone is NOT proof of unbound.
//   - a present-but-non-null association, including an empty object "{}", -> false.
//   - a staticIp:null with a NON-null vpc/privateNetwork (a contradiction) -> false.
//   - a non-object / malformed body -> false (fail closed).
//
// Mutation that turns this RED: relax the predicate to accept omitted fields (e.g.
// decode into the struct and test nil pointers) -> the "omitted" cases flip to true.
func TestVPCFloatingIPBodyProvesFullyUnbound(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{"all three present and explicitly null is the only proof", `{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`, true},
		{"a body that OMITS all associations is NOT proof (the blocker case)", `{"id":"fip-1"}`, false},
		{"a body that omits only staticIp is NOT proof", `{"id":"fip-1","vpc":null,"privateNetwork":null}`, false},
		{"a body that omits only vpc is NOT proof", `{"id":"fip-1","staticIp":null,"privateNetwork":null}`, false},
		{"a body that omits only privateNetwork is NOT proof", `{"id":"fip-1","staticIp":null,"vpc":null}`, false},
		{"a bound staticIp (present, non-null) is not unbound", `{"id":"fip-1","staticIp":{"id":"si-1","address":"10.0.1.5"},"vpc":null,"privateNetwork":null}`, false},
		{"an empty staticIp object {} (present, non-null) is not unbound", `{"id":"fip-1","staticIp":{},"vpc":null,"privateNetwork":null}`, false},
		{"a non-null vpc with staticIp null is a contradiction -> not unbound", `{"id":"fip-1","staticIp":null,"vpc":{"id":"v1"},"privateNetwork":null}`, false},
		{"a non-object body fails closed", `"a string, not an object"`, false},
		{"a JSON null body fails closed", `null`, false},
		{"a malformed body fails closed", `{not json`, false},
		// Duplicate-key contradictions: Go's map decode keeps the LAST value, so a body
		// that FIRST declares a binding then re-declares the key null would collapse to
		// null and falsely look unbound. The raw-token walk must reject ANY duplicated
		// top-level key (not just associations) as structurally untrustworthy.
		{"a duplicate staticIp (bound object THEN null) is rejected (last-wins would falsely look unbound)", `{"id":"fip-1","staticIp":{"id":"si-1"},"staticIp":null,"vpc":null,"privateNetwork":null}`, false},
		{"a duplicate staticIp (null THEN bound object) is rejected", `{"id":"fip-1","staticIp":null,"staticIp":{"id":"si-1"},"vpc":null,"privateNetwork":null}`, false},
		{"a duplicate of a benign key is also rejected (any duplicate top-level key is untrustworthy)", `{"id":"fip-1","id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`, false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := floatingIPBodyProvesFullyUnbound([]byte(tc.body)); got != tc.want {
				t.Fatalf("floatingIPBodyProvesFullyUnbound(%s) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

// TestVPCFloatingIPDeprovisionUnbound pins the ONLY exported deletion path: gated
// AND confirmed by construction. A billable IP is deleted ONLY when provably
// fully-unbound, and success ALWAYS rests on a POSITIVE final 404 — never a bare
// 2xx. The gate failures issue NO DELETE at all (a `deleted` flag proves it).
func TestVPCFloatingIPDeprovisionUnbound(t *testing.T) {
	ctx := context.Background()

	t.Run("an already-absent (404 on resolve) FIP is an idempotent success with NO DELETE", func(t *testing.T) {
		var deleted bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleted = true
			}
			w.WriteHeader(http.StatusNotFound)
		})
		if err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-gone", nil); err != nil {
			t.Fatalf("an authoritative 404 on resolve must be an idempotent success, got %v", err)
		}
		if deleted {
			t.Fatal("an already-absent FIP must not be DELETEd")
		}
	})

	t.Run("a FIP bound to a static IP gets a typed refusal with NO DELETE", func(t *testing.T) {
		var deleted bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleted = true
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":{"id":"si-1","address":"10.0.1.5"}}`))
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		if err == nil {
			t.Fatal("a bound FIP must be refused, never deprovisioned")
		}
		if !strings.Contains(err.Error(), "si-1") {
			t.Fatalf("the refusal must name the binding static IP, got %v", err)
		}
		if deleted {
			t.Fatal("a bound FIP must NOT be DELETEd (gated by construction)")
		}
	})

	t.Run("a contradictory association state (staticIp null, vpc set) fails closed with NO DELETE", func(t *testing.T) {
		var deleted bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleted = true
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":{"id":"v1"}}`))
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		if err == nil {
			t.Fatal("a contradictory association state must fail closed, never deprovision")
		}
		if !strings.Contains(err.Error(), "inconclusive") {
			t.Fatalf("the refusal must flag the inconclusive association state, got %v", err)
		}
		if deleted {
			t.Fatal("an inconclusive FIP must NOT be DELETEd")
		}
	})

	t.Run("a body that OMITS the association fields is NOT proof of unbound: fail closed with NO DELETE (the #310 destructive blocker)", func(t *testing.T) {
		// Go decodes an OMITTED json pointer to nil, identically to an explicit null.
		// A destructive delete must therefore rest on POSITIVE present-and-null evidence
		// recovered from the raw body, never on absence-of-fields. A detail endpoint that
		// returns only {"id":...} (associations omitted) must be treated as inconclusive,
		// because a billable FIP that is actually bound would otherwise be silently deleted.
		var deleted bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleted = true
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fip-1"}`))
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		// The rock-solid, stub-independent invariant FIRST: no DELETE may be issued on an
		// unproven body, whatever a subsequent confirm read would have returned.
		if deleted {
			t.Fatal("a billable FIP whose unbound state is unproven must NOT be DELETEd")
		}
		if err == nil {
			t.Fatal("a body that omits the association fields must fail closed; omitted is not proof of unbound")
		}
		if !strings.Contains(err.Error(), "inconclusive") {
			t.Fatalf("the refusal must flag the inconclusive association state, got %v", err)
		}
	})

	t.Run("an id-less staticIp object {} is NOT a nameable binding NOR proof of unbound: inconclusive, NO DELETE", func(t *testing.T) {
		// staticIp is PRESENT but its raw value is {} (not null), so it is not proof of
		// unbound; yet it carries no id, so it cannot yield a typed "bound to si-X" refusal
		// either. The only safe classification is inconclusive -> fail closed, no DELETE.
		var deleted bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleted = true
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":{},"vpc":null,"privateNetwork":null}`))
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		if err == nil {
			t.Fatal("an id-less staticIp object must fail closed, never deprovision")
		}
		if !strings.Contains(err.Error(), "inconclusive") {
			t.Fatalf("an id-less staticIp object must be classed inconclusive (no nameable binding), got %v", err)
		}
		if deleted {
			t.Fatal("an inconclusive FIP must NOT be DELETEd")
		}
	})

	t.Run("a duplicate-key contradiction body (bound staticIp THEN staticIp:null) fails closed with NO DELETE", func(t *testing.T) {
		// Go's json decode keeps the LAST value for a duplicate key, so the map proof
		// would see staticIp:null and FALSELY green-light deleting a FIP the same body
		// declares bound to si-1. The gate must reject the duplicated key outright. This
		// is the round-2 destructive blocker: a billable, possibly-bound IP must NEVER be
		// deleted on a contradictory body.
		var deleted bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleted = true
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":{"id":"si-1"},"staticIp":null,"vpc":null,"privateNetwork":null}`))
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		// Rock-solid invariant FIRST: a contradictory body must issue NO DELETE.
		if deleted {
			t.Fatal("a duplicate-key contradiction must NOT be DELETEd (last-wins null must not green-light a destructive delete)")
		}
		if err == nil {
			t.Fatal("a duplicate-key contradiction body must fail closed, never deprovision")
		}
		if !strings.Contains(err.Error(), "inconclusive") {
			t.Fatalf("a duplicate-key contradiction must be classed inconclusive, got %v", err)
		}
	})

	t.Run("a fully-unbound FIP whose raw DELETE is already gone (404) confirms via a final 404", func(t *testing.T) {
		var getCount int
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				getCount++
				if getCount == 1 {
					// First resolve: present + fully unbound.
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`))
					return
				}
				// Confirm read-back: authoritative absent.
				w.WriteHeader(http.StatusNotFound)
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.WriteHeader(http.StatusNotFound) // raw says already gone
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		if err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil); err != nil {
			t.Fatalf("a fully-unbound FIP that is already gone must confirm and succeed, got %v", err)
		}
	})

	t.Run("a fully-unbound FIP with an async DELETE waits the activity then confirms via a final 404", func(t *testing.T) {
		var getCount int
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				getCount++
				if getCount == 1 {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.Header().Set("Location", "act-del")
				w.WriteHeader(http.StatusOK)
			case r.Method == http.MethodGet && r.URL.Path == "/activity/v1/activities/act-del":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(completedActivityBody("fip-1")))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		if err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil); err != nil {
			t.Fatalf("a fully-unbound async deprovision that completes and confirms 404 must succeed, got %v", err)
		}
	})

	t.Run("a deprovision whose final read-back is STILL present fails closed (no false success)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				// BOTH the initial resolve and the confirm read-back return present.
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`))
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.WriteHeader(http.StatusNoContent) // sync 2xx, no Location
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		if err == nil {
			t.Fatal("a deprovision whose read-back is still present must fail closed (a bare 2xx is never proof of removal)")
		}
		if !strings.Contains(err.Error(), "still present") {
			t.Fatalf("the failure must flag the still-present read-back, got %v", err)
		}
	})

	t.Run("a fully-unbound FIP whose deprovision activity FAILS surfaces an error carrying the activityID", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`))
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.Header().Set("Location", "act-del")
				w.WriteHeader(http.StatusOK)
			case r.Method == http.MethodGet && r.URL.Path == "/activity/v1/activities/act-del":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"act-del","state":{"failed":{}}}`))
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		if err == nil {
			t.Fatal("a failed deprovision activity must surface as an error, never a silent success")
		}
		if !strings.Contains(err.Error(), "act-del") {
			t.Fatalf("the failure must carry the activityID for correlation, got %v", err)
		}
	})

	t.Run("a fully-unbound FIP whose raw DELETE is 403 fails closed (forbidden is not deleted)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`))
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.WriteHeader(http.StatusForbidden)
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		if err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil); err == nil {
			t.Fatal("a 403 on the raw DELETE must fail closed, never report deprovisioned")
		}
	})

	t.Run("a 403 on the INITIAL gate read fails closed with NO DELETE (forbidden is not absence, #303)", func(t *testing.T) {
		var deleted bool
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleted = true
			}
			// The gate read itself is DENIED: this is NOT an authoritative 404, so the
			// destructive path must refuse rather than treat "can't read it" as "gone".
			w.WriteHeader(http.StatusForbidden)
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		if err == nil {
			t.Fatal("a 403 on the initial resolve must fail closed (forbidden is never absence), not silently succeed")
		}
		if deleted {
			t.Fatal("a denied gate read must NEVER reach the DELETE (the gate could not prove the FIP is safe to delete)")
		}
	})

	t.Run("a 403 on the positive-confirmation read fails closed (denied is not proof of removal)", func(t *testing.T) {
		var getCount int
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				getCount++
				if getCount == 1 {
					// Gate read: present + fully unbound, so the DELETE is allowed to run.
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"id":"fip-1","staticIp":null,"vpc":null,"privateNetwork":null}`))
					return
				}
				// Confirm read-back: DENIED. A 403 is NOT a 404, so removal is unproven.
				w.WriteHeader(http.StatusForbidden)
			case r.Method == http.MethodDelete && r.URL.Path == "/vpc/v1/floating_ips/fip-1":
				w.WriteHeader(http.StatusNoContent) // sync 2xx, no Location
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
		err := c.VPC().FloatingIP().DeprovisionUnbound(ctx, "fip-1", nil)
		if err == nil {
			t.Fatal("a 403 on the confirm read-back must fail closed: success ALWAYS rests on a positive 404, never on a denied confirm")
		}
		if getCount != 2 {
			t.Fatalf("expected exactly 2 by-id reads (gate + confirm), got %d", getCount)
		}
	})
}
