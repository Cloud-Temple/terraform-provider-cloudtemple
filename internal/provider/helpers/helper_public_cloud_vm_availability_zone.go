package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMAvailabilityZone maps a client.PublicCloudVMAvailabilityZone
// to the flat snake_case map consumed by both the single and list datasources.
func FlattenPublicCloudVMAvailabilityZone(az *client.PublicCloudVMAvailabilityZone) map[string]interface{} {
	families := make([]map[string]interface{}, len(az.CompatibleFamilies))
	for i, f := range az.CompatibleFamilies {
		families[i] = map[string]interface{}{
			"id":   f.ID,
			"name": f.Name,
		}
	}
	return map[string]interface{}{
		"id":                  az.ID,
		"name":                az.Name,
		"description":         az.Description,
		"region_id":           az.RegionID,
		"is_enabled":          az.IsEnabled,
		"compatible_families": families,
		"created_at":          az.CreatedAt,
		"updated_at":          az.UpdatedAt,
	}
}
