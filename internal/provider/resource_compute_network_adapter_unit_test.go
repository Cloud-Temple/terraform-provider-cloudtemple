package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestVMwareVPCStaticIPToPush pins the VMware update decision for the VPC static
// IP (ip_address, #375): never push when unconfigured/empty/off-VPC, never
// re-push an unchanged address (it would relocate the static IP to itself on
// every apply), push on a genuine divergence or a first set.
func TestVMwareVPCStaticIPToPush(t *testing.T) {
	cases := []struct {
		name                 string
		ipConfigured         bool
		configuredIP, liveIP string
		onVPC                bool
		want                 string
	}{
		{"not configured -> never push", false, "", "", true, ""},
		{"configured but empty -> never push", true, "", "", true, ""},
		{"configured non-empty but not on a VPC network -> never push", true, "10.0.2.10", "", false, ""},
		{"configured, on VPC, no live IP yet -> push (first set)", true, "10.0.2.10", "", true, "10.0.2.10"},
		{"configured equals live -> no redundant relocate-to-self", true, "10.0.2.10", "10.0.2.10", true, ""},
		{"configured diverges from live -> relocate", true, "10.0.2.11", "10.0.2.10", true, "10.0.2.11"},
		{"stale configured value with ipConfigured=false (field cleared) -> never push", false, "10.0.2.11", "10.0.2.10", true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := vmwareVPCStaticIPToPush(c.ipConfigured, c.configuredIP, c.liveIP, c.onVPC); got != c.want {
				t.Fatalf("vmwareVPCStaticIPToPush(%v,%q,%q,%v) = %q, want %q", c.ipConfigured, c.configuredIP, c.liveIP, c.onVPC, got, c.want)
			}
		})
	}
}

// TestVMwareAdapterVPCStaticIP pins the read mapping for ip_address (#375): a
// non-VPC adapter and a VPC adapter without a registered static IP both yield
// empty; a VPC adapter reflects the by-MAC static IP address.
func TestVMwareAdapterVPCStaticIP(t *testing.T) {
	if got := vmwareAdapterVPCStaticIP(false, &client.StaticIP{IPAddress: "1.2.3.4"}); got != "" {
		t.Fatalf("a non-VPC adapter has no static IP; want empty, got %q", got)
	}
	if got := vmwareAdapterVPCStaticIP(true, nil); got != "" {
		t.Fatalf("a VPC adapter with no registered static IP (nil) must yield empty, got %q", got)
	}
	if got := vmwareAdapterVPCStaticIP(true, &client.StaticIP{IPAddress: "10.0.2.10"}); got != "10.0.2.10" {
		t.Fatalf("a VPC adapter must reflect the by-MAC static IP; want 10.0.2.10, got %q", got)
	}
}
