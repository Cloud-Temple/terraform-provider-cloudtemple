package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

func TestVIFCleanupTargets(t *testing.T) {
	listed := map[string]bool{"vif-ours": true}

	t.Run("nil failed activity cleans nothing", func(t *testing.T) {
		toDelete, unconfirmed := vifCleanupTargets(nil, listed)
		if len(toDelete) != 0 || len(unconfirmed) != 0 {
			t.Fatalf("toDelete=%v unconfirmed=%v, want none", toDelete, unconfirmed)
		}
	})

	t.Run("referenced and listed on this VM is deleted", func(t *testing.T) {
		failed := &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
			{ID: "vif-ours", Type: "network_adapter"},
		}}
		toDelete, unconfirmed := vifCleanupTargets(failed, listed)
		if len(toDelete) != 1 || toDelete[0] != "vif-ours" || len(unconfirmed) != 0 {
			t.Fatalf("toDelete=%v unconfirmed=%v, want [vif-ours]/none", toDelete, unconfirmed)
		}
	})

	t.Run("referenced but absent from the strict listing is UNCONFIRMED (forbids the retry)", func(t *testing.T) {
		// By attribution the ConcernedItems of OUR create are ours: an
		// absence from the listing may be eventual consistency right after
		// the incident — never a green light to retry (would duplicate).
		failed := &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
			{ID: "vif-laggy", Type: "network_adapter"},
		}}
		toDelete, unconfirmed := vifCleanupTargets(failed, listed)
		if len(toDelete) != 0 || len(unconfirmed) != 1 || unconfirmed[0] != "vif-laggy" {
			t.Fatalf("toDelete=%v unconfirmed=%v, want none/[vif-laggy]", toDelete, unconfirmed)
		}
	})

	t.Run("non-adapter items and empty ids are ignored", func(t *testing.T) {
		failed := &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
			{ID: "vif-ours", Type: "virtual_machine"},
			{ID: "", Type: "network_adapter"},
		}}
		toDelete, unconfirmed := vifCleanupTargets(failed, listed)
		if len(toDelete) != 0 || len(unconfirmed) != 0 {
			t.Fatalf("toDelete=%v unconfirmed=%v, want none", toDelete, unconfirmed)
		}
	})
}

func TestClassifyMissingDevice(t *testing.T) {
	scoped := map[string]bool{"on-vm": true}
	tenant := map[string]bool{"on-vm": true, "elsewhere": true}

	if got := classifyMissingDevice("gone", scoped, tenant); got != deviceDeletionConfirmed {
		t.Fatalf("absent everywhere must confirm the deletion, got %v", got)
	}
	if got := classifyMissingDevice("on-vm", scoped, tenant); got != deviceStillInScope {
		t.Fatalf("still in the scoped listing must fail closed (access restriction), got %v", got)
	}
	if got := classifyMissingDevice("elsewhere", scoped, tenant); got != deviceExistsOutOfScope {
		t.Fatalf("present tenant-wide only is drift (detached/moved), never a deletion, got %v", got)
	}
}

// TestVPCStaticIPToPush pins the update decision for the VPC static IP
// (ip_address, #374). The subtle rules: never push when unconfigured/empty/off-VPC,
// never re-push an unchanged address (it would relocate the static IP to itself
// on every apply), push on a genuine divergence or a first set.
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

// TestAdapterVPCStaticIP pins the read mapping for ip_address (#374): a non-VPC
// adapter and a VPC adapter without a registered static IP both yield empty; a
// VPC adapter reflects the by-MAC static IP address.
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
