package helpers

import (
	"sort"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMInstanceFamily maps a client.PublicCloudVMInstanceFamily to
// the flat snake_case map consumed by both the single and list datasources.
func FlattenPublicCloudVMInstanceFamily(family *client.PublicCloudVMInstanceFamily) map[string]interface{} {
	return map[string]interface{}{
		"id":          family.ID,
		"name":        family.Name,
		"description": family.Description,
		"vcpu_min":    family.VcpuMin,
		"vcpu_max":    family.VcpuMax,
		"ram_min_gb":  family.RamMinGb,
		"ram_max_gb":  family.RamMaxGb,
		"skus":        flattenPublicCloudVMSkus(family.Skus),
	}
}

// flattenPublicCloudVMSkus maps the priced SKU catalogue of an instance family.
// It returns a non-nil (possibly empty) slice so a family with no SKUs pins the
// "present but empty" intent for the Terraform state writer rather than dropping
// the key. The SKUs are sorted by their stable `name` key: the API does not
// guarantee a deterministic order, and a read-only datasource whose elements can
// be indexed (skus[0]) must not surface a spurious reorder as downstream churn.
func flattenPublicCloudVMSkus(skus []client.PublicCloudVMSku) []map[string]interface{} {
	sorted := make([]client.PublicCloudVMSku, len(skus))
	copy(sorted, skus)
	// SliceStable so SKUs sharing a name (should not happen — name is the SKU id —
	// but we do not assume it) keep their relative API order deterministically.
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	out := make([]map[string]interface{}, len(sorted))
	for i, s := range sorted {
		out[i] = map[string]interface{}{
			"name":           s.Name,
			"price":          s.Price,
			"unit":           s.Unit,
			"description":    s.Description,
			"description_en": s.DescriptionEn,
		}
	}
	return out
}
