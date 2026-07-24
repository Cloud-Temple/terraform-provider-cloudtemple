package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMImage maps a client.PublicCloudVMImage to the flat
// snake_case map consumed by both the single and list datasources. Slice fields
// are normalized to non-nil empty slices.
func FlattenPublicCloudVMImage(img *client.PublicCloudVMImage) map[string]interface{} {
	diskSizes := make([]interface{}, len(img.DiskSizesGb))
	for i, s := range img.DiskSizesGb {
		diskSizes[i] = s
	}
	families := make([]interface{}, len(img.CompatibleFamilies))
	for i, f := range img.CompatibleFamilies {
		families[i] = f
	}
	categories := make([]interface{}, len(img.Categories))
	for i, c := range img.Categories {
		categories[i] = c
	}
	return map[string]interface{}{
		"id":                  img.ID,
		"name":                img.Name,
		"os_family":           img.OsFamily,
		"os_name":             img.OsName,
		"os_version":          img.OsVersion,
		"disk_sizes_gb":       diskSizes,
		"compatible_families": families,
		"categories":          categories,
		"family":              img.Family,
		"version":             img.Version,
		"editor":              img.Editor,
		"description_en":      img.DescriptionEn,
		"image_type":          img.ImageType,
		"icon":                img.Icon,
	}
}
