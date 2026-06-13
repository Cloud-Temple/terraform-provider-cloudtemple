package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

func osLister(names []string, err error) func(context.Context) ([]string, error) {
	return func(context.Context) ([]string, error) {
		return names, err
	}
}

// TestConfirmObjectStorageOrKeep pins the nil-read handler: it ALWAYS fails
// closed (never a drop, never a silent stale-state success) and distinguishes
// "still listed" from "absent" only to sharpen the diagnostic (#281).
func TestConfirmObjectStorageOrKeep(t *testing.T) {
	ctx := context.Background()
	const name = "bucket-1"

	t.Run("a list error fails closed", func(t *testing.T) {
		diags := confirmObjectStorageOrKeep(ctx, name, "object storage bucket", osLister(nil, errors.New("boom")))
		if !diags.HasError() {
			t.Fatal("a list error must fail closed")
		}
	})

	t.Run("a still-listed resource fails closed (no stale success)", func(t *testing.T) {
		diags := confirmObjectStorageOrKeep(ctx, name, "object storage bucket", osLister([]string{"other", name}, nil))
		if !diags.HasError() {
			t.Fatal("a still-listed resource must fail closed, never report a successful refresh")
		}
		if !strings.Contains(diags[0].Summary, "still listed") {
			t.Fatalf("must take the still-listed branch, got %q", diags[0].Summary)
		}
	})

	t.Run("an absent resource fails closed (never drops)", func(t *testing.T) {
		diags := confirmObjectStorageOrKeep(ctx, name, "object storage bucket", osLister([]string{"other"}, nil))
		if !diags.HasError() {
			t.Fatal("an unconfirmed absence must fail closed, never auto-remove the resource")
		}
		if !strings.Contains(diags[0].Summary, "not in the listing") {
			t.Fatalf("must take the not-listed branch, got %q", diags[0].Summary)
		}
	})

	t.Run("an empty listing takes the absent branch", func(t *testing.T) {
		diags := confirmObjectStorageOrKeep(ctx, name, "object storage bucket", osLister(nil, nil))
		if !diags.HasError() || !strings.Contains(diags[0].Summary, "not in the listing") {
			t.Fatalf("empty listing must be treated as absent, got %v", diags)
		}
	})

	t.Run("substring and superstring names do not count as present", func(t *testing.T) {
		diags := confirmObjectStorageOrKeep(ctx, name, "object storage bucket", osLister([]string{"bucket-12", "bucket-", "bucket"}, nil))
		if !diags.HasError() || !strings.Contains(diags[0].Summary, "not in the listing") {
			t.Fatalf("near-misses must be treated as absent, got %v", diags)
		}
	})

	t.Run("empty entries are skipped without hiding a present name", func(t *testing.T) {
		diags := confirmObjectStorageOrKeep(ctx, name, "object storage bucket", osLister([]string{"", name, ""}, nil))
		if !strings.Contains(diags[0].Summary, "still listed") {
			t.Fatalf("a present name must be found despite empty entries, got %q", diags[0].Summary)
		}
	})
}

// TestObjectStorageNameProjections proves a JSON `[null]` entry (decoded into a
// nil pointer) is skipped before reading the name, so the evidence listing
// never panics and never injects an empty name.
func TestObjectStorageNameProjections(t *testing.T) {
	t.Run("bucketNames skips nil entries", func(t *testing.T) {
		got := bucketNames([]*client.Bucket{{Name: "a"}, nil, {Name: "b"}})
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("nil bucket entries must be skipped, got %v", got)
		}
	})

	t.Run("storageAccountNames skips nil entries", func(t *testing.T) {
		got := storageAccountNames([]*client.StorageAccount{nil, {Name: "x"}, nil})
		if len(got) != 1 || got[0] != "x" {
			t.Fatalf("nil account entries must be skipped, got %v", got)
		}
	})
}
