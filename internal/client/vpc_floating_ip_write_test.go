package client

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// TestVPCFloatingIPBind pins the ASYNCHRONOUS bind contract: a POST to
// /floating_ips/{fip}/bind/static_ips/{static} that returns an activity
// (Location). An empty/sync body is tolerated (the id is read from Location). A
// 409 Conflict is an idempotent non-error (already bound to the same pair), and
// returns an empty activity id so the resource layer confirms by read.
func TestVPCFloatingIPBind(t *testing.T) {
	ctx := context.Background()

	t.Run("async bind hits the right path and returns the activity id from Location", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vpc/v1/floating_ips/fip-1/bind/static_ips/si-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Location", "act-bind")
			w.WriteHeader(http.StatusCreated)
		})
		activityID, err := c.VPC().FloatingIP().Bind(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if activityID != "act-bind" {
			t.Fatalf("the async bind must return the activity id from Location, got %q", activityID)
		}
	})

	// The live API may answer 200/201 WITHOUT a body. doRequestAndReturnActivity
	// reads only the Location header, so an empty body must not be a failure as
	// long as the Location is present.
	t.Run("an empty body with a Location is tolerated", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "act-x")
			w.WriteHeader(http.StatusOK) // empty body
		})
		activityID, err := c.VPC().FloatingIP().Bind(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("an empty body with a Location must be tolerated: %v", err)
		}
		if activityID != "act-x" {
			t.Fatalf("activity id must come from Location, got %q", activityID)
		}
	})

	// A 409 Conflict is the live "already bound to the SAME static IP" signal. It
	// must be a non-error idempotent success returning an empty activity id (there
	// is no activity to wait on); the resource layer confirms the pair by read.
	t.Run("a 409 Conflict is an idempotent success with no activity id", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
		})
		activityID, err := c.VPC().FloatingIP().Bind(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("a 409 must be an idempotent success, got error: %v", err)
		}
		if activityID != "" {
			t.Fatalf("a 409 has no activity to wait on; the id must be empty, got %q", activityID)
		}
	})

	t.Run("a bind with no Location is an error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated) // no Location header
		})
		if _, err := c.VPC().FloatingIP().Bind(ctx, "fip-1", "si-1"); err == nil {
			t.Fatal("an async bind with no Location header must be an error")
		}
	})

	t.Run("a 403 on the bind call surfaces as an error", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		if _, err := c.VPC().FloatingIP().Bind(ctx, "fip-1", "si-1"); err == nil {
			t.Fatal("a 403 on bind must surface as an error, not be swallowed")
		}
	})
}

// TestVPCFloatingIPUnbind pins the ASYNCHRONOUS unbind contract: a DELETE to
// /floating_ips/{fip}/unbind/static_ips/{static} that returns an activity
// (Location), with the same empty-body and 409-idempotent tolerances as Bind.
func TestVPCFloatingIPUnbind(t *testing.T) {
	ctx := context.Background()

	t.Run("async unbind hits the right path and returns the activity id from Location", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/vpc/v1/floating_ips/fip-1/unbind/static_ips/si-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Location", "act-unbind")
			w.WriteHeader(http.StatusCreated)
		})
		activityID, err := c.VPC().FloatingIP().Unbind(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if activityID != "act-unbind" {
			t.Fatalf("the async unbind must return the activity id from Location, got %q", activityID)
		}
	})

	t.Run("a 409 Conflict is an idempotent success with no activity id", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
		})
		activityID, err := c.VPC().FloatingIP().Unbind(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("a 409 must be an idempotent success, got error: %v", err)
		}
		if activityID != "" {
			t.Fatalf("a 409 has no activity to wait on; the id must be empty, got %q", activityID)
		}
	})

	// A 404/403 on the unbind CALL must surface as an error (a StatusError),
	// because the resource layer applies a strict positive-confirmation rule
	// before treating it as idempotent — it must not be swallowed here.
	t.Run("a 404 surfaces as a StatusError{404}", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		_, err := c.VPC().FloatingIP().Unbind(ctx, "fip-1", "si-gone")
		if err == nil {
			t.Fatal("a 404 on unbind must surface as an error so the resource can confirm")
		}
		var statusErr StatusError
		if !errors.As(err, &statusErr) || statusErr.Code != http.StatusNotFound {
			t.Fatalf("a 404 must be a StatusError{404}, got %v", err)
		}
	})

	t.Run("a 403 surfaces as a StatusError{403}", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		_, err := c.VPC().FloatingIP().Unbind(ctx, "fip-1", "si-1")
		if err == nil {
			t.Fatal("a 403 on unbind must surface as an error")
		}
		var statusErr StatusError
		if !errors.As(err, &statusErr) || statusErr.Code != http.StatusForbidden {
			t.Fatalf("a 403 must be a StatusError{403}, got %v", err)
		}
	})
}

// TestVPCFloatingIPListStrict pins the corroboration channel: ONLY a complete
// HTTP 200 array is usable. A 206 is partial; 201/403/5xx are not evidence; a
// malformed/null/object 200, or a 200 with an id-less entry, is an error.
func TestVPCFloatingIPListStrict(t *testing.T) {
	ctx := context.Background()

	t.Run("200 hits /vpc/v1/floating_ips with no vpcId filter and returns the parsed list", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/vpc/v1/floating_ips" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			if r.URL.Query().Get("vpcId") != "" {
				t.Errorf("ListStrict must not send a vpcId filter, got %q", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id":"fip-1","ipAddress":"198.51.100.1","staticIp":{"id":"si-1","address":"10.0.1.5"}},
				{"id":"fip-2","ipAddress":"198.51.100.2","staticIp":null}
			]`))
		})
		list, err := c.VPC().FloatingIP().ListStrict(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 || list[0].ID != "fip-1" || list[1].ID != "fip-2" {
			t.Fatalf("unexpected list: %+v", list)
		}
	})

	t.Run("200 with malformed JSON errors", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		})
		if _, err := c.VPC().FloatingIP().ListStrict(ctx); err == nil {
			t.Fatal("a malformed 200 body must return a decode error")
		}
	})

	t.Run("200 with a null body is rejected", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`null`))
		})
		if _, err := c.VPC().FloatingIP().ListStrict(ctx); err == nil {
			t.Fatal("a 200 null body must be rejected, never read as a (proven) empty listing")
		}
	})

	t.Run("200 with a JSON object (not an array) is rejected", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"floating_ips":[]}`))
		})
		if _, err := c.VPC().FloatingIP().ListStrict(ctx); err == nil {
			t.Fatal("a 200 object body must be rejected: only a JSON array can corroborate")
		}
	})

	t.Run("200 with an entry missing its id is rejected (structurally incomplete)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"fip-1"},{"ipAddress":"198.51.100.9"}]`))
		})
		if _, err := c.VPC().FloatingIP().ListStrict(ctx); err == nil {
			t.Fatal("a listing with an id-less entry must be rejected as structurally incomplete")
		}
	})

	t.Run("200 with an empty JSON array is a valid listing", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		})
		list, err := c.VPC().FloatingIP().ListStrict(ctx)
		if err != nil {
			t.Fatalf("an empty JSON array is a valid listing, must not error: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected an empty list, got %d entries", len(list))
		}
	})

	for _, code := range []int{
		http.StatusCreated,
		http.StatusPartialContent,
		http.StatusForbidden,
		http.StatusInternalServerError,
	} {
		code := code
		t.Run(http.StatusText(code)+" is rejected as non-200 evidence", func(t *testing.T) {
			c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`[]`))
			})
			if _, err := c.VPC().FloatingIP().ListStrict(ctx); err == nil {
				t.Fatalf("status %d must FAIL CLOSED: only a 200 is corroboration", code)
			}
		})
	}
}

// TestVPCFloatingIPCorroborateBinding pins the strict FOUR-STATE classification
// used by the resource layer. The anti-clobber invariant is the whole point of
// the four states: "present & unbound" (safe to bind) MUST be distinct from
// "present & bound to a DIFFERENT static IP" (must fail closed, never bind). The
// overriding state-safety invariant also holds: only a POSITIVELY OBSERVED
// floating IP yields a definite state; everything structurally unprovable (FIP
// absent from the listing, null/non-array/id-less body) is Inconclusive — never
// negative evidence.
func TestVPCFloatingIPCorroborateBinding(t *testing.T) {
	ctx := context.Background()

	t.Run("present and bound to the target static IP -> BoundToTarget", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"fip-1","staticIp":{"id":"si-1","address":"10.0.1.5"}}]`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != FloatingIPBindingBoundToTarget {
			t.Fatalf("a FIP bound to the target static IP must be BoundToTarget, got %v", state)
		}
	})

	t.Run("present and UNBOUND (staticIp nil) -> Unbound (the only bind-unlocking state)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"fip-1","staticIp":null}]`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != FloatingIPBindingUnbound {
			t.Fatalf("a present FIP with staticIp nil must be Unbound, got %v", state)
		}
	})

	t.Run("present but bound to a DIFFERENT static IP -> BoundToOther (NOT Unbound)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"fip-1","staticIp":{"id":"si-OTHER","address":"10.0.1.9"}}]`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != FloatingIPBindingBoundToOther {
			t.Fatalf("a FIP bound to a different static IP must be BoundToOther (never collapsed with Unbound), got %v", state)
		}
		// Anti-clobber contract: this state must be distinct from Unbound, so the
		// create path can never bind on a FIP bound elsewhere.
		if FloatingIPBindingBoundToOther == FloatingIPBindingUnbound {
			t.Fatal("BoundToOther and Unbound must be DISTINCT states (anti-clobber)")
		}
	})

	// A present FIP whose nested staticIp is an OBJECT but with an empty/omitted id
	// is STRUCTURALLY INCONCLUSIVE — the API returned a staticIp but omitted its id,
	// which is neither proof of unbound nor of bound-elsewhere. It must classify as
	// Inconclusive, NEVER as BoundToOther (the #312 R6 lesson applied to the nested
	// id). Treating it as BoundToOther would let the resource layer use it as
	// negative evidence (drop state / accept an unbind) while the FIP may still be
	// bound to OUR static IP.
	t.Run("present but nested staticIp is an empty object {} -> Inconclusive (NOT BoundToOther)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"fip-1","staticIp":{}}]`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != FloatingIPBindingInconclusive {
			t.Fatalf("a present FIP with an id-less nested staticIp must be Inconclusive, never BoundToOther, got %v", state)
		}
	})

	t.Run("present but nested staticIp has an address and no id -> Inconclusive (NOT BoundToOther)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"fip-1","staticIp":{"address":"10.0.1.5"}}]`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != FloatingIPBindingInconclusive {
			t.Fatalf("a nested staticIp with an address but no id must be Inconclusive, never BoundToOther, got %v", state)
		}
	})

	t.Run("the FIP is NOT in the listing -> Inconclusive (never negative evidence)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"fip-OTHER","staticIp":null}]`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != FloatingIPBindingInconclusive {
			t.Fatalf("a FIP absent from the listing must be Inconclusive, never read as absent/negative, got %v", state)
		}
	})

	t.Run("a structurally incomplete (id-less entry) listing -> Inconclusive", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"fip-1","staticIp":null},{"staticIp":null}]`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err == nil {
			t.Fatal("a listing with an id-less entry must surface the ListStrict error")
		}
		if state != FloatingIPBindingInconclusive {
			t.Fatalf("a structurally incomplete listing must classify as Inconclusive, got %v", state)
		}
	})

	t.Run("a 200 null body -> Inconclusive (never a proven empty listing)", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`null`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err == nil {
			t.Fatal("a 200 null body must surface the ListStrict error")
		}
		if state != FloatingIPBindingInconclusive {
			t.Fatalf("a null body must classify as Inconclusive, got %v", state)
		}
	})

	t.Run("a 200 JSON object (not an array) -> Inconclusive", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"floating_ips":[]}`))
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err == nil {
			t.Fatal("a 200 object body must surface the ListStrict error")
		}
		if state != FloatingIPBindingInconclusive {
			t.Fatalf("a non-array body must classify as Inconclusive, got %v", state)
		}
	})

	t.Run("an inconclusive (non-200) listing surfaces the error and is Inconclusive", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		state, err := c.VPC().FloatingIP().CorroborateBinding(ctx, "fip-1", "si-1")
		if err == nil {
			t.Fatal("a non-200 listing must surface the ListStrict error")
		}
		if state != FloatingIPBindingInconclusive {
			t.Fatalf("a failed listing must classify as Inconclusive, got %v", state)
		}
	})
}
