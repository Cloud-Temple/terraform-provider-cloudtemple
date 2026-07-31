package helpers

import (
	"math"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// TestFlattenPublicCloudVMStorageTypeSku is the regression proof for issue #507.
// It goes RED on the pre-#507 helper (which emitted no "sku" key at all) and
// GREEN once the helper maps the nested billing object. It pins the exact
// nested shape the datasource schema expects: a one-element list of a
// snake_case map when a SKU is present.
func TestFlattenPublicCloudVMStorageTypeSku(t *testing.T) {
	st := &client.PublicCloudVMStorageType{
		ID:          "st-1",
		Name:        "Standard",
		Description: "Standard block storage",
		IopsHint:    "~1500 IOPS/TB",
		MinSizeGb:   1,
		MaxSizeGb:   2048,
		IsAvailable: true,
		Sku: &client.PublicCloudVMSku{
			Name:          "csp:fr1:iaas:storage:bloc:medium:v1",
			Price:         0.063,
			Unit:          "1 Gio",
			Description:   "Stockage bloc medium",
			DescriptionEn: "Medium block storage",
		},
	}

	out := FlattenPublicCloudVMStorageType(st)

	raw, ok := out["sku"]
	if !ok {
		t.Fatalf("flatten emits no %q key; the datasource cannot expose the SKU", "sku")
	}
	sku, ok := raw.([]map[string]interface{})
	if !ok {
		t.Fatalf("sku has type %T, want []map[string]interface{} (nested list-of-object shape)", raw)
	}
	if len(sku) != 1 {
		t.Fatalf("want a one-element sku list, got %d elements", len(sku))
	}
	e := sku[0]
	if e["name"] != "csp:fr1:iaas:storage:bloc:medium:v1" || e["unit"] != "1 Gio" {
		t.Fatalf("sku string fields not flattened: %+v", e)
	}
	if e["description"] != "Stockage bloc medium" || e["description_en"] != "Medium block storage" {
		t.Fatalf("sku description pair not flattened (fr/en): %+v", e)
	}
	price, ok := e["price"].(float64)
	if !ok {
		t.Fatalf("sku price has type %T, want float64", e["price"])
	}
	if math.Abs(price-0.063) > 1e-9 {
		t.Fatalf("sku price not flattened: got %v, want ~0.063", price)
	}
}

// TestFlattenPublicCloudVMStorageTypeSkuNil pins that a nil Sku (the API omitted
// it or sent null) flattens to a NON-NIL empty list, not a nil value or a
// phantom zero-priced element. This is what keeps the generic zero-input
// invariant (helper_flatten_invariants_test.go) green without a knownNilSliceGaps
// entry, and what stops the state from carrying a bogus price=0 SKU.
func TestFlattenPublicCloudVMStorageTypeSkuNil(t *testing.T) {
	out := FlattenPublicCloudVMStorageType(&client.PublicCloudVMStorageType{ID: "st-nosku"})

	raw, ok := out["sku"]
	if !ok {
		t.Fatalf("flatten must always emit the %q key, even when nil", "sku")
	}
	sku, ok := raw.([]map[string]interface{})
	if !ok {
		t.Fatalf("sku has type %T, want []map[string]interface{}", raw)
	}
	if sku == nil {
		t.Fatalf("sku is a typed nil slice on nil Sku; want a non-nil empty list to pin the present-but-empty intent")
	}
	if len(sku) != 0 {
		t.Fatalf("want an empty sku list on nil Sku, got %d elements: %+v", len(sku), sku)
	}
}
