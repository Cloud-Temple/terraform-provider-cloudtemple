package helpers

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestUpdateMapItemsDoesNotRetainExplicitZeroValues PINS the known
// limitation of the GetOk-based merge (#246 class): GetOk cannot
// distinguish an explicit zero value (false, "", 0) from an absent
// attribute, so an explicit user false is NOT overlaid onto the API-seeded
// map and the live value silently wins.
//
// This is why any boolean whose explicit false must reach an API request
// is gated on raw-config evidence (d.GetRawConfig()) instead of trusting
// the merged map — see osAdapterTxConfigured and the
// TestOptionalComputedBooleansHavePatchGuards registry in the provider
// package. If this test ever fails because UpdateMapItems starts retaining
// explicit zero values, revisit every raw-config gating site before
// relying on the new behavior.
func TestUpdateMapItemsDoesNotRetainExplicitZeroValues(t *testing.T) {
	s := map[string]*schema.Schema{
		"os_network_adapter": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"enabled": {Type: schema.TypeBool, Optional: true},
					"name":    {Type: schema.TypeString, Optional: true},
				},
			},
		},
	}
	d := schema.TestResourceDataRaw(t, s, map[string]interface{}{
		"os_network_adapter": []interface{}{
			map[string]interface{}{
				"enabled": false,       // explicit user false
				"name":    "user-name", // explicit non-zero value
			},
		},
	})

	apiSeeded := map[string]interface{}{
		"enabled": true, // live API value
		"name":    "api-name",
	}
	merged, ok := UpdateMapItems(d, apiSeeded, "os_network_adapter", 0).(map[string]interface{})
	if !ok {
		t.Fatal("UpdateMapItems did not return a map")
	}

	// Non-zero explicit values are overlaid onto the API-seeded map.
	if merged["name"] != "user-name" {
		t.Errorf("explicit non-zero value lost: name=%v, want user-name", merged["name"])
	}

	// Explicit zero values are swallowed: the API-seeded value wins. This
	// assertion documents the CURRENT behavior, it does not endorse it.
	if merged["enabled"] != true {
		t.Errorf("the GetOk swallowing behavior changed: enabled=%v (was: API value retained) — revisit the raw-config gating sites before relying on this", merged["enabled"])
	}
}

// TestUpdateNestedMapItemsPreservesPerIndexMapping pins that
// UpdateNestedMapItems overlays the config onto EACH element using that
// element's own index, never crossing indices, and preserves the element count
// and order. It also confirms the same explicit-zero swallowing limitation
// (#246 class) propagates per element: an explicit user false at one index is
// swallowed there, while a non-zero override at another index is applied.
//
// Crossing indices (e.g. using index 0's config for element 1) would corrupt
// multi-element nested blocks (multiple network adapters / disks) — a real
// state hazard the single-index UpdateMapItems test cannot catch.
func TestUpdateNestedMapItemsPreservesPerIndexMapping(t *testing.T) {
	s := map[string]*schema.Schema{
		"os_network_adapter": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"enabled": {Type: schema.TypeBool, Optional: true},
					"name":    {Type: schema.TypeString, Optional: true},
				},
			},
		},
	}
	d := schema.TestResourceDataRaw(t, s, map[string]interface{}{
		"os_network_adapter": []interface{}{
			// index 0: explicit false + explicit name
			map[string]interface{}{"enabled": false, "name": "user-0"},
			// index 1: explicit true + explicit name
			map[string]interface{}{"enabled": true, "name": "user-1"},
		},
	})

	apiSeeded := []interface{}{
		map[string]interface{}{"enabled": true, "name": "api-0"},
		map[string]interface{}{"enabled": false, "name": "api-1"},
	}

	out := UpdateNestedMapItems(d, apiSeeded, "os_network_adapter")
	if len(out) != 2 {
		t.Fatalf("UpdateNestedMapItems returned %d elements, want 2 (count must be preserved)", len(out))
	}

	e0, ok := out[0].(map[string]interface{})
	if !ok {
		t.Fatalf("element 0 has type %T, want map[string]interface{}", out[0])
	}
	e1, ok := out[1].(map[string]interface{})
	if !ok {
		t.Fatalf("element 1 has type %T, want map[string]interface{}", out[1])
	}

	// Non-zero overrides are applied at the CORRECT index (no cross-talk).
	if e0["name"] != "user-0" {
		t.Errorf("element 0 name = %v, want user-0 (config index 0 must overlay element 0)", e0["name"])
	}
	if e1["name"] != "user-1" {
		t.Errorf("element 1 name = %v, want user-1 (config index 1 must overlay element 1)", e1["name"])
	}

	// index 1 has an explicit user true -> non-zero -> overlaid onto the
	// API-seeded false. This proves a per-index NON-zero boolean override does
	// reach the merged map.
	if e1["enabled"] != true {
		t.Errorf("element 1 enabled = %v, want true (explicit non-zero override at index 1 must be applied)", e1["enabled"])
	}

	// index 0 has an explicit user false -> zero -> swallowed by GetOk, so the
	// API-seeded true wins. Same #246 limitation as UpdateMapItems, pinned per
	// element. This documents the CURRENT behavior; it does not endorse it.
	if e0["enabled"] != true {
		t.Errorf("element 0 enabled = %v; the per-index GetOk swallowing behavior changed (was: API true retained) — revisit the raw-config gating sites before relying on this", e0["enabled"])
	}
}

// TestUpdateNestedMapItemsEmptyInputIsEmptyOutput pins that an empty nested
// input yields a non-nil empty slice (stable list shape), not a nil.
func TestUpdateNestedMapItemsEmptyInputIsEmptyOutput(t *testing.T) {
	s := map[string]*schema.Schema{
		"os_network_adapter": {
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Resource{Schema: map[string]*schema.Schema{"name": {Type: schema.TypeString, Optional: true}}},
		},
	}
	d := schema.TestResourceDataRaw(t, s, map[string]interface{}{})
	out := UpdateNestedMapItems(d, []interface{}{}, "os_network_adapter")
	if out == nil {
		t.Errorf("UpdateNestedMapItems returned nil; expected a non-nil empty slice")
	}
	if len(out) != 0 {
		t.Errorf("UpdateNestedMapItems = %v, want empty", out)
	}
}
