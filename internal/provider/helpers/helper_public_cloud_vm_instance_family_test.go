package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestFlattenPublicCloudVMInstanceFamilyEmitsSkus is the offline completeness
// guard the structural walker cannot provide: the walker proves that whatever
// the helper EMITS fits the schema, but it does not fail if the helper simply
// FORGETS to emit a declared Computed attribute (a never-Set attribute is just
// empty). This test pins that `skus` is actually emitted, with the exact nested
// key set and values, and sorted by the stable `name` key. It fails RED on a
// FlattenPublicCloudVMInstanceFamily that does not emit `skus` (the pre-#506
// helper).
func TestFlattenPublicCloudVMInstanceFamilyEmitsSkus(t *testing.T) {
	// SKUs are supplied in API order (vcpu then ram); the helper must return
	// them sorted by name (ram sorts before vcpu), independent of input order.
	family := &client.PublicCloudVMInstanceFamily{
		ID:   "fam-1",
		Name: "General Purpose",
		Skus: []client.PublicCloudVMSku{
			{Name: "csp:fr1:vminstance:gp:vcpu:v1", Price: 23.42, Unit: "vcpu", Description: "Coeur FR", DescriptionEn: "Core EN"},
			{Name: "csp:fr1:vminstance:gp:ram:v1", Price: 2, Unit: "gio", Description: "RAM FR", DescriptionEn: "RAM EN"},
		},
	}

	out := FlattenPublicCloudVMInstanceFamily(family)

	raw, present := out["skus"]
	if !present {
		t.Fatalf("FlattenPublicCloudVMInstanceFamily emits no `skus` key; got keys %v", keysOf(out))
	}
	skus, ok := raw.([]map[string]interface{})
	if !ok {
		t.Fatalf("`skus` is %T, want []map[string]interface{}", raw)
	}
	if len(skus) != 2 {
		t.Fatalf("want 2 skus emitted, got %d (%+v)", len(skus), skus)
	}

	// Sorted by name: "...:ram:v1" < "...:vcpu:v1", so ram is first.
	if got := skus[0]["name"]; got != "csp:fr1:vminstance:gp:ram:v1" {
		t.Errorf("skus not sorted by name: skus[0].name = %v, want the ram sku first", got)
	}
	if got := skus[1]["name"]; got != "csp:fr1:vminstance:gp:vcpu:v1" {
		t.Errorf("skus not sorted by name: skus[1].name = %v, want the vcpu sku second", got)
	}

	// Exact nested key set (no extra, no missing) and values on the vcpu sku.
	vcpu := skus[1]
	wantKeys := map[string]interface{}{
		"name":           "csp:fr1:vminstance:gp:vcpu:v1",
		"price":          23.42,
		"unit":           "vcpu",
		"description":    "Coeur FR",
		"description_en": "Core EN",
	}
	if len(vcpu) != len(wantKeys) {
		t.Errorf("sku key set = %v, want exactly %v", keysOf(vcpu), keysOf(wantKeys))
	}
	for k, want := range wantKeys {
		if got := vcpu[k]; got != want {
			t.Errorf("sku[%q] = %v (%T), want %v (%T)", k, got, got, want, want)
		}
	}
	// price must be a float64 (TypeFloat), including the integer-valued one.
	if _, isFloat := skus[0]["price"].(float64); !isFloat {
		t.Errorf("ram sku price is %T, want float64", skus[0]["price"])
	}
}

// TestFlattenPublicCloudVMInstanceFamilyEmptySkusIsNonNil pins the "present but
// empty" intent (invariant iii of the flatten harness) at the unit level: a
// family with no SKUs must still emit `skus` as a non-nil empty slice, never a
// typed nil (which would let the SDK silently drop the key).
func TestFlattenPublicCloudVMInstanceFamilyEmptySkusIsNonNil(t *testing.T) {
	out := FlattenPublicCloudVMInstanceFamily(&client.PublicCloudVMInstanceFamily{ID: "fam-2"})
	raw, present := out["skus"]
	if !present {
		t.Fatalf("`skus` key absent on a family with no skus")
	}
	skus, ok := raw.([]map[string]interface{})
	if !ok {
		t.Fatalf("`skus` is %T, want []map[string]interface{}", raw)
	}
	if skus == nil {
		t.Errorf("`skus` is a typed nil slice; want a non-nil empty slice ([]) to pin present-but-empty intent")
	}
	if len(skus) != 0 {
		t.Errorf("want 0 skus, got %d", len(skus))
	}
}
