package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

func TestVIFCleanupTargets(t *testing.T) {
	listed := map[string]bool{"vif-ours": true}

	t.Run("nil failed activity cleans nothing", func(t *testing.T) {
		if got := vifCleanupTargets(nil, listed); len(got) != 0 {
			t.Fatalf("targets=%v, want none", got)
		}
	})

	t.Run("referenced and listed on this VM is deleted", func(t *testing.T) {
		failed := &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
			{ID: "vif-ours", Type: "network_adapter"},
		}}
		got := vifCleanupTargets(failed, listed)
		if len(got) != 1 || got[0] != "vif-ours" {
			t.Fatalf("targets=%v, want [vif-ours]", got)
		}
	})

	t.Run("referenced but absent from the strict listing is never deleted", func(t *testing.T) {
		failed := &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
			{ID: "vif-elsewhere", Type: "network_adapter"},
		}}
		if got := vifCleanupTargets(failed, listed); len(got) != 0 {
			t.Fatalf("targets=%v, want none (gone, or not ours to delete)", got)
		}
	})

	t.Run("non-adapter items and empty ids are ignored", func(t *testing.T) {
		failed := &client.Activity{ConcernedItems: []client.ActivityConcernedItem{
			{ID: "vif-ours", Type: "virtual_machine"},
			{ID: "", Type: "network_adapter"},
		}}
		if got := vifCleanupTargets(failed, listed); len(got) != 0 {
			t.Fatalf("targets=%v, want none", got)
		}
	})
}
