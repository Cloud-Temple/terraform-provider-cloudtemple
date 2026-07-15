package helpers

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestGetStringList pins GetStringList (#293, S7). It reads a Terraform list
// attribute into a []string for the datasource filter payloads. It must
// preserve order and content, and must fail closed to a non-nil empty slice —
// never nil, never a panic — when the key is unset or not a list. Callers feed
// the result straight into API filter structs, so a wrong order, a dropped
// element, or a nil return would silently change what gets queried.
// Non-complacent: dropping the copied value reds the order case; returning nil
// on the not-a-list path reds the non-list case.
func TestGetStringList(t *testing.T) {
	sm := map[string]*schema.Schema{
		"tags": {
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"name": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}

	t.Run("a configured list returns its strings in order", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, sm, map[string]interface{}{
			"tags": []interface{}{"x", "y", "z"},
		})
		got := GetStringList(d, "tags")
		want := []string{"x", "y", "z"}
		if len(got) != len(want) {
			t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("element %d = %q, want %q (order/copy bug): %#v", i, got[i], want[i], got)
			}
		}
	})

	t.Run("an empty list returns a non-nil empty slice", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, sm, map[string]interface{}{
			"tags": []interface{}{},
		})
		got := GetStringList(d, "tags")
		if got == nil {
			t.Fatal("got nil, want a non-nil empty slice")
		}
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0: %#v", len(got), got)
		}
	})

	t.Run("an unset list key returns a non-nil empty slice", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, sm, map[string]interface{}{})
		got := GetStringList(d, "tags")
		if got == nil || len(got) != 0 {
			t.Fatalf("got %#v, want a non-nil empty slice", got)
		}
	})

	t.Run("a non-list key returns a non-nil empty slice (never panics)", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, sm, map[string]interface{}{
			"name": "web",
		})
		got := GetStringList(d, "name")
		if got == nil || len(got) != 0 {
			t.Fatalf("got %#v, want a non-nil empty slice", got)
		}
	})
}
