package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

// TestUpdateStaticIPRequestJSONFieldSet pins that the PATCH body can NEVER carry
// ipAddress (the swagger UpdateStaticIpPayload does not allow it) — only
// macAddress and resourceDescription, and only when set.
func TestUpdateStaticIPRequestJSONFieldSet(t *testing.T) {
	mac := "00:50:56:ab:cd:ef"
	desc := "d"
	cases := map[string]struct {
		req  client.UpdateStaticIPRequest
		keys []string
	}{
		"mac only":         {client.UpdateStaticIPRequest{MacAddress: &mac}, []string{"macAddress"}},
		"description only": {client.UpdateStaticIPRequest{ResourceDescription: &desc}, []string{"resourceDescription"}},
		"both":             {client.UpdateStaticIPRequest{MacAddress: &mac, ResourceDescription: &desc}, []string{"macAddress", "resourceDescription"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b, err := json.Marshal(tc.req)
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
			if len(m) != len(tc.keys) {
				t.Fatalf("unexpected key set %s, want %v", b, tc.keys)
			}
			for _, k := range tc.keys {
				if _, ok := m[k]; !ok {
					t.Fatalf("missing key %q in %s", k, b)
				}
			}
		})
	}
}
