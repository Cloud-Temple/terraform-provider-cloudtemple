package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMRegion maps a client.PublicCloudVMRegion to the flat
// snake_case map consumed by both the single and list datasources.
func FlattenPublicCloudVMRegion(region *client.PublicCloudVMRegion) map[string]interface{} {
	return map[string]interface{}{
		"id":           region.ID,
		"name":         region.Name,
		"description":  region.Description,
		"country_code": region.CountryCode,
		"geography":    region.Geography,
		"is_enabled":   region.IsEnabled,
		"az_count":     region.AzCount,
		"created_at":   region.CreatedAt,
		"updated_at":   region.UpdatedAt,
	}
}
