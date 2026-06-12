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
