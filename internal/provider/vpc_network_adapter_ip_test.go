package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestVPCStaticIPToPush pins the update decision for the VPC static IP
// (ip_address) shared by both adapter resources (#374/#375, unified in #379).
// The subtle rules: never push when unconfigured/empty/off-VPC, never re-push an
// unchanged address (it would relocate the static IP to itself on every apply),
// push on a genuine divergence or a first set.
func TestVPCStaticIPToPush(t *testing.T) {
	cases := []struct {
		name                 string
		ipConfigured         bool
		configuredIP, liveIP string
		onVPC                bool
		want                 string
	}{
		{"not configured -> never push", false, "", "", true, ""},
		{"configured but empty -> never push", true, "", "", true, ""},
		{"configured non-empty but not on a VPC network -> never push", true, "192.168.0.10", "", false, ""},
		{"configured, on VPC, no live IP yet -> push (first set)", true, "192.168.0.10", "", true, "192.168.0.10"},
		{"configured equals live -> no redundant relocate-to-self", true, "192.168.0.10", "192.168.0.10", true, ""},
		{"configured diverges from live -> relocate", true, "192.168.0.11", "192.168.0.10", true, "192.168.0.11"},
		{"a stale configured value with ipConfigured=false (field cleared) -> never push", false, "192.168.0.11", "192.168.0.10", true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := vpcStaticIPToPush(c.ipConfigured, c.configuredIP, c.liveIP, c.onVPC); got != c.want {
				t.Fatalf("vpcStaticIPToPush(%v,%q,%q,%v) = %q, want %q", c.ipConfigured, c.configuredIP, c.liveIP, c.onVPC, got, c.want)
			}
		})
	}
}

// TestAdapterVPCStaticIP pins the read mapping for ip_address shared by both
// adapter resources (#374/#375, unified in #379): a non-VPC adapter and a VPC
// adapter without a registered static IP both yield empty; a VPC adapter reflects
// the by-MAC static IP address.
func TestAdapterVPCStaticIP(t *testing.T) {
	if got := adapterVPCStaticIP(false, &client.StaticIP{IPAddress: "1.2.3.4"}); got != "" {
		t.Fatalf("a non-VPC adapter has no static IP; want empty, got %q", got)
	}
	if got := adapterVPCStaticIP(true, nil); got != "" {
		t.Fatalf("a VPC adapter with no registered static IP (nil) must yield empty, got %q", got)
	}
	if got := adapterVPCStaticIP(true, &client.StaticIP{IPAddress: "192.168.0.10"}); got != "192.168.0.10" {
		t.Fatalf("a VPC adapter must reflect the by-MAC static IP; want 192.168.0.10, got %q", got)
	}
}

// TestValidateIPAddressTargetsVPC pins the fail-closed verdict of the shared
// ip_address pre-validation (#379): once the target network has been read,
// ip_address is accepted ONLY on an existing, VPC-backed network. A network that
// does not exist must be rejected (never silently accepted as if it were a VPC
// network — that would let a side effect run against an unverified target), and a
// network that exists but is not VPC-backed must be rejected (ip_address has no
// meaning there and the plan would never converge). This guards the contract for
// BOTH adapter resources at once, since they share this function.
func TestValidateIPAddressTargetsVPC(t *testing.T) {
	t.Run("network not found -> fail closed", func(t *testing.T) {
		d := validateIPAddressTargetsVPC("192.168.0.10", "net-1", false, false)
		if !d.HasError() {
			t.Fatal("a network that does not exist must be rejected, got no error")
		}
	})
	t.Run("found but not VPC-backed -> reject", func(t *testing.T) {
		d := validateIPAddressTargetsVPC("192.168.0.10", "net-1", false, true)
		if !d.HasError() {
			t.Fatal("ip_address on a non-VPC network must be rejected, got no error")
		}
	})
	t.Run("found and VPC-backed -> accept", func(t *testing.T) {
		// Defensive: vpcBacked is only meaningful when found; a VPC-backed,
		// existing network is the one case that must pass.
		if d := validateIPAddressTargetsVPC("192.168.0.10", "net-1", true, true); d.HasError() {
			t.Fatalf("ip_address on an existing VPC network must pass, got %v", d)
		}
	})
}
