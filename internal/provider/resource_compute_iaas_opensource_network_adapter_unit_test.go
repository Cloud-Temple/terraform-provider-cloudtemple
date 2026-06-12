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
