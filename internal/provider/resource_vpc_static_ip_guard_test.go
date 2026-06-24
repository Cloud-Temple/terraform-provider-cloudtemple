package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestVPCStaticIPSourceGuard pins the #311 guard: this resource manages only
// "custom" static IPs, and admits one only on POSITIVE proof (source=="custom").
// A non-custom (platform-managed, e.g. "xoa") static IP read — which happens on
// `terraform import` or a refresh of a wrongly-adopted state — must be REJECTED,
// because such a static IP cannot be deleted via the API, so a TF-managed resource
// pointing at it would be undeletable. An EMPTY source is likewise rejected: it is
// not proof of custom ownership (a custom static IP always reports source="custom",
// live contract), so tolerating it would be fail-OPEN — a malformed read could
// silently adopt an undeletable IP. Every rejection KEEPS the id (never a drop).
//
// Non-complacent: without the guard the non-custom/empty read would succeed and
// flatten. The reads use readForRefresh (the refresh/import path), but the source
// guard fires on the returned object regardless of mode — it is reached before any
// listing.
func TestVPCStaticIPSourceGuard(t *testing.T) {
	ctx := context.Background()

	t.Run("a non-custom (xoa) static IP is rejected", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d,
			siRead(&client.StaticIP{ID: "si-1", Source: "xoa", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			siListStrict(), readForRefresh)
		if !diags.HasError() {
			t.Fatal("a non-custom (xoa) static IP must be rejected; it cannot be managed/deleted via Terraform")
		}
		// Rejecting must NOT drop the state: a wrongly-adopted resource is surfaced
		// as an error for the operator to `terraform state rm`, never silently removed.
		if d.Id() != "si-1" {
			t.Fatalf("a rejected non-custom read must keep the id for the operator to resolve, got %q", d.Id())
		}
	})

	t.Run("a custom static IP is accepted", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d,
			siRead(&client.StaticIP{ID: "si-1", Source: "custom", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			siListStrict(), readForRefresh)
		if diags.HasError() {
			t.Fatalf("a custom static IP must be accepted, got: %v", diags)
		}
	})

	t.Run("an empty source fails closed (an unproven source is never adopted)", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d,
			siRead(&client.StaticIP{ID: "si-1", Source: "", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			siListStrict(), readForRefresh)
		if !diags.HasError() {
			t.Fatal("an empty source is not positive proof of a custom static IP; it must FAIL CLOSED, never be adopted (fail-open would let a malformed read adopt an undeletable IP)")
		}
		// Fail closed must KEEP the id: an inconclusive read is never a drop.
		if d.Id() != "si-1" {
			t.Fatalf("an empty-source fail-closed read must keep the id, got %q", d.Id())
		}
	})
}

// TestIsVPCStatusCode pins the status-code predicate used by delete to route
// 404 (unambiguous absence) and 403 (ambiguous, must be confirmed) differently.
func TestIsVPCStatusCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code int
		want bool
	}{
		{"404 matches 404", client.StatusError{Code: http.StatusNotFound}, http.StatusNotFound, true},
		{"403 matches 403", client.StatusError{Code: http.StatusForbidden}, http.StatusForbidden, true},
		{"403 does NOT match 404 (the two codes must not be conflated)", client.StatusError{Code: http.StatusForbidden}, http.StatusNotFound, false},
		{"500 matches neither", client.StatusError{Code: http.StatusInternalServerError}, http.StatusNotFound, false},
		{"a non-status error never matches", errors.New("boom"), http.StatusForbidden, false},
		{"a WRAPPED status error still matches (errors.As unwrap)", errors.Join(errors.New("ctx"), client.StatusError{Code: http.StatusNotFound}), http.StatusNotFound, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isVPCStatusCode(tc.err, tc.code); got != tc.want {
				t.Fatalf("isVPCStatusCode(%v, %d) = %v, want %v", tc.err, tc.code, got, tc.want)
			}
		})
	}
}

// TestDeleteVPCStaticIPWith pins the delete state-safety contract. The overriding
// invariant: the resource is NEVER dropped without positive evidence the static
// IP is gone. In particular a 403 (ambiguous under #303) must be CONFIRMED via a
// strict listing — a permission failure must fail closed, not silently drop the
// state. Non-complacent: a 403-as-absent shortcut reds the "still listed",
// "listing fails" and "missing scope" cases below.
//
// There is NO retry on the write path (the transient-502 retry was retired — the
// platform bug it guarded is fixed); a failed delete activity therefore fails
// closed in a SINGLE pass, which the "failed delete activity" case pins.
func TestDeleteVPCStaticIPWith(t *testing.T) {
	ctx := context.Background()
	okWait := func(ctx context.Context, activityID string) error { return nil }
	noDelete := func(ctx context.Context, id string) (string, error) {
		return "", errors.New("delete must not be called")
	}
	noList := func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error) {
		return nil, errors.New("listing must not be reached")
	}
	noWait := func(ctx context.Context, activityID string) error {
		return errors.New("wait must not be reached")
	}
	delErr := func(err error) vpcStaticIPDeleteFunc {
		return func(ctx context.Context, id string) (string, error) { return "", err }
	}

	t.Run("a completed activity deletes cleanly", func(t *testing.T) {
		d := newStaticIPState(t)
		del := func(ctx context.Context, id string) (string, error) { return "act-1", nil }
		diags := deleteVPCStaticIPWith(ctx, d, del, noList, okWait)
		if diags.HasError() {
			t.Fatalf("a completed delete activity must succeed, got: %v", diags)
		}
	})

	t.Run("a 404 is an idempotent success (already gone)", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := deleteVPCStaticIPWith(ctx, d, delErr(client.StatusError{Code: http.StatusNotFound}), noList, noWait)
		if diags.HasError() {
			t.Fatalf("a 404 delete must be idempotent success, got: %v", diags)
		}
	})

	t.Run("a 403 confirmed absent by a strict listing is an idempotent success", func(t *testing.T) {
		d := newStaticIPState(t)
		list := siListStrict(&client.StaticIP{ID: "other"}) // si-1 absent
		diags := deleteVPCStaticIPWith(ctx, d, delErr(client.StatusError{Code: http.StatusForbidden}), list, noWait)
		if diags.HasError() {
			t.Fatalf("a 403 whose absence is confirmed must succeed, got: %v", diags)
		}
	})

	t.Run("a 403 but STILL LISTED fails closed (a forbidden delete must not drop state)", func(t *testing.T) {
		d := newStaticIPState(t)
		list := siListStrict(&client.StaticIP{ID: "si-1"}, &client.StaticIP{ID: "other"})
		diags := deleteVPCStaticIPWith(ctx, d, delErr(client.StatusError{Code: http.StatusForbidden}), list, noWait)
		if !diags.HasError() {
			t.Fatal("a 403 on a still-present static IP must fail closed, never be read as a deletion")
		}
	})

	t.Run("a 403 with a failing strict listing fails closed", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := deleteVPCStaticIPWith(ctx, d, delErr(client.StatusError{Code: http.StatusForbidden}), siListStrictErr(errors.New("403")), noWait)
		if !diags.HasError() {
			t.Fatal("a 403 with an inconclusive listing must fail closed")
		}
	})

	t.Run("a 403 with a missing private_network_id fails closed", func(t *testing.T) {
		d := newStaticIPState(t)
		if err := d.Set("private_network_id", ""); err != nil {
			t.Fatalf("clearing private_network_id: %v", err)
		}
		diags := deleteVPCStaticIPWith(ctx, d, delErr(client.StatusError{Code: http.StatusForbidden}), noList, noWait)
		if !diags.HasError() {
			t.Fatal("a 403 without a scope to confirm absence must fail closed")
		}
	})

	t.Run("a 500 delete is an error", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := deleteVPCStaticIPWith(ctx, d, delErr(client.StatusError{Code: http.StatusInternalServerError}), noList, noWait)
		if !diags.HasError() {
			t.Fatal("a 500 must surface as a delete error")
		}
	})

	t.Run("a failed delete activity is an error (a failed activity is not deletion evidence, no retry)", func(t *testing.T) {
		d := newStaticIPState(t)
		del := func(ctx context.Context, id string) (string, error) { return "act-1", nil }
		failWait := func(ctx context.Context, activityID string) error { return errors.New("activity failed") }
		diags := deleteVPCStaticIPWith(ctx, d, del, noList, failWait)
		if !diags.HasError() {
			t.Fatal("a failed delete activity must surface as an error in a single pass, not a silent success")
		}
	})

	t.Run("a non-custom source is rejected before any API call", func(t *testing.T) {
		d := newStaticIPState(t)
		if err := d.Set("source", "xoa"); err != nil {
			t.Fatalf("seeding source: %v", err)
		}
		diags := deleteVPCStaticIPWith(ctx, d, noDelete, noList, noWait)
		if !diags.HasError() {
			t.Fatal("a non-custom static IP must be rejected by the delete preflight")
		}
	})
}

// TestStaticIPDeleteErrorDetail pins that the platform's "not a custom static IP"
// refusal becomes an actionable diagnostic, while a generic error passes through.
func TestStaticIPDeleteErrorDetail(t *testing.T) {
	t.Run("not-a-custom refusal maps to an actionable diagnostic", func(t *testing.T) {
		got := staticIPDeleteErrorDetail(errors.New(`Static IP x is not a custom static IP (source: xoa)`))
		if !strings.Contains(got, "custom") || !strings.Contains(got, "terraform state rm") {
			t.Fatalf("expected an actionable not-custom message, got %q", got)
		}
	})
	t.Run("a generic error is returned verbatim", func(t *testing.T) {
		if got := staticIPDeleteErrorDetail(errors.New("boom")); got != "boom" {
			t.Fatalf("expected the raw error, got %q", got)
		}
	})
}

// TestUpdateStaticIPRequestFieldSet pins, STRUCTURALLY, that the PATCH body type
// carries ONLY resourceDescription and macAddress — and can NEVER carry ipAddress
// (the swagger UpdateStaticIpPayload). The reflection check fails if ANY field is
// added, including an `omitempty` one, so it is NOT complacent against a future
// ipAddress field (which a marshal-only test would miss when the field is unset).
// The marshaling check additionally pins the diff-driven (omitempty) body.
func TestUpdateStaticIPRequestFieldSet(t *testing.T) {
	want := map[string]bool{"resourceDescription": true, "macAddress": true}
	typ := reflect.TypeOf(client.UpdateStaticIPRequest{})
	got := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if name == "ipAddress" {
			t.Fatal("UpdateStaticIPRequest must never carry ipAddress (not in UpdateStaticIpPayload)")
		}
		got[name] = true
	}
	if len(got) != len(want) {
		t.Fatalf("UpdateStaticIPRequest json field set = %v, want exactly %v (any added field is forbidden)", got, want)
	}
	for k := range want {
		if !got[k] {
			t.Fatalf("missing json field %q", k)
		}
	}

	// Runtime: a diff-driven (omitempty) body emits only the set fields, never ipAddress.
	mac := "00:50:56:ab:cd:ef"
	b, err := json.Marshal(client.UpdateStaticIPRequest{MacAddress: &mac})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, bad := m["ipAddress"]; bad {
		t.Fatalf("the PATCH body must NEVER contain ipAddress, got %s", b)
	}
	if len(m) != 1 || m["macAddress"] == nil {
		t.Fatalf("a mac-only PATCH body must be exactly {macAddress}, got %s", b)
	}
}
