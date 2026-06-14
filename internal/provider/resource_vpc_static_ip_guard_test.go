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
// "custom" static IPs. A non-custom (platform-managed, e.g. "xoa") static IP read
// — which happens on `terraform import` or a refresh of a wrongly-adopted state —
// must be REJECTED, because such a static IP cannot be deleted via the API, so a
// TF-managed resource pointing at it would be undeletable.
//
// Non-complacent: without the guard the non-custom read would succeed.
func TestVPCStaticIPSourceGuard(t *testing.T) {
	ctx := context.Background()

	t.Run("a non-custom (xoa) static IP is rejected", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d,
			siRead(&client.StaticIP{ID: "si-1", Source: "xoa", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			siListStrict())
		if !diags.HasError() {
			t.Fatal("a non-custom (xoa) static IP must be rejected; it cannot be managed/deleted via Terraform")
		}
	})

	t.Run("a custom static IP is accepted", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d,
			siRead(&client.StaticIP{ID: "si-1", Source: "custom", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			siListStrict())
		if diags.HasError() {
			t.Fatalf("a custom static IP must be accepted, got: %v", diags)
		}
	})

	t.Run("an empty source is tolerated (no spurious failure on a transient read)", func(t *testing.T) {
		d := newStaticIPState(t)
		diags := readVPCStaticIPInto(ctx, d,
			siRead(&client.StaticIP{ID: "si-1", Source: "", PrivateNetwork: client.BaseObject{ID: "pn-1"}}, nil),
			siListStrict())
		if diags.HasError() {
			t.Fatalf("an empty source must not be rejected, got: %v", diags)
		}
	})
}

// TestIsVPCStaticIPAbsent pins the idempotent-delete predicate: the VPC API
// reports absence as 404 OR 403 (#303 403-as-not-found), so both make delete
// idempotent. Non-complacent: a 404-only implementation reds the 403 case.
func TestIsVPCStaticIPAbsent(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"404 is absent", client.StatusError{Code: http.StatusNotFound}, true},
		{"403 is absent (VPC 403-as-not-found, #303)", client.StatusError{Code: http.StatusForbidden}, true},
		{"500 is not absent", client.StatusError{Code: http.StatusInternalServerError}, false},
		{"a non-status error is not absent", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isVPCStaticIPAbsent(tc.err); got != tc.want {
				t.Fatalf("isVPCStaticIPAbsent(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
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
