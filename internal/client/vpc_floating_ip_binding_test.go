package client

import (
	"context"
	"net/http"
	"testing"
)

// serveFIPBody returns a GET handler for /vpc/v1/floating_ips/{wantID} that writes
// the given status and RAW body. Raw bodies let the tests inject shapes the decoded
// struct cannot represent — staticIp OMITTED vs explicit null, and duplicate
// top-level keys — which is exactly what ResolveBinding must classify correctly.
func serveFIPBody(t *testing.T, wantID, body string, status int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vpc/v1/floating_ips/"+wantID {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}
}

// TestResolveBinding pins the binding resource's authoritative by-id oracle. The
// load-bearing distinctions — which the LISTING oracle cannot make and which a naive
// decoded-struct read gets WRONG — are: staticIp OMITTED (structurally incomplete ->
// Inconclusive) vs explicit null (-> Unbound), and a duplicate-key body that
// last-wins-decodes to a FALSE Unbound. Each case asserts the (state, found, err,
// fip-non-nil) tuple a mutant would break.
func TestResolveBinding(t *testing.T) {
	ctx := context.Background()
	const fip, target = "fip-1", "si-1"

	cases := []struct {
		name      string
		status    int
		body      string
		wantState FloatingIPBindingState
		wantFound bool
		wantErr   bool
		wantFIP   bool
	}{
		{
			name:   "404 is authoritative absent (the sole drop signal)",
			status: http.StatusNotFound, body: "",
			wantState: FloatingIPBindingInconclusive, wantFound: false, wantErr: false, wantFIP: false,
		},
		{
			name:   "403 fails closed (forbidden is not absence, #303)",
			status: http.StatusForbidden, body: "",
			wantState: FloatingIPBindingInconclusive, wantFound: false, wantErr: true, wantFIP: false,
		},
		{
			name:   "staticIp OMITTED is Inconclusive, NEVER Unbound",
			status: http.StatusOK, body: `{"id":"fip-1","ipAddress":"198.51.100.7"}`,
			wantState: FloatingIPBindingInconclusive, wantFound: true, wantErr: false, wantFIP: true,
		},
		{
			name:   "staticIp explicit null is Unbound (free to bind)",
			status: http.StatusOK, body: `{"id":"fip-1","ipAddress":"198.51.100.7","staticIp":null,"vpc":null,"privateNetwork":null}`,
			wantState: FloatingIPBindingUnbound, wantFound: true, wantErr: false, wantFIP: true,
		},
		{
			name:   "staticIp bound to OUR target",
			status: http.StatusOK, body: `{"id":"fip-1","staticIp":{"id":"si-1","address":"10.0.1.5"}}`,
			wantState: FloatingIPBindingBoundToTarget, wantFound: true, wantErr: false, wantFIP: true,
		},
		{
			name:   "staticIp bound to a DIFFERENT static IP (anti-clobber evidence)",
			status: http.StatusOK, body: `{"id":"fip-1","staticIp":{"id":"si-2","address":"10.0.1.9"}}`,
			wantState: FloatingIPBindingBoundToOther, wantFound: true, wantErr: false, wantFIP: true,
		},
		{
			name:   "staticIp object with an EMPTY id is Inconclusive (structurally incomplete)",
			status: http.StatusOK, body: `{"id":"fip-1","staticIp":{"address":"10.0.1.9"}}`,
			wantState: FloatingIPBindingInconclusive, wantFound: true, wantErr: false, wantFIP: true,
		},
		{
			name:   "DUPLICATE top-level staticIp (object then null) is Inconclusive, never last-wins Unbound",
			status: http.StatusOK, body: `{"id":"fip-1","staticIp":{"id":"si-1"},"staticIp":null}`,
			wantState: FloatingIPBindingInconclusive, wantFound: true, wantErr: false, wantFIP: false,
		},
		{
			name:   "mismatched top-level id fails closed (id guard)",
			status: http.StatusOK, body: `{"id":"someone-else","staticIp":null}`,
			wantState: FloatingIPBindingInconclusive, wantFound: true, wantErr: true, wantFIP: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newVPCTestClient(t, serveFIPBody(t, fip, tc.body, tc.status))
			state, gotFIP, found, err := c.VPC().FloatingIP().ResolveBinding(ctx, fip, target)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if state != tc.wantState {
				t.Fatalf("state = %v, want %v", state, tc.wantState)
			}
			if found != tc.wantFound {
				t.Fatalf("found = %v, want %v", found, tc.wantFound)
			}
			if (gotFIP != nil) != tc.wantFIP {
				t.Fatalf("fip non-nil = %v, want %v", gotFIP != nil, tc.wantFIP)
			}
		})
	}
}
