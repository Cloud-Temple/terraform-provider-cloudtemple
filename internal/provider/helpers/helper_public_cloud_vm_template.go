package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMTemplate maps a client.PublicCloudVMTemplate to the flat
// snake_case map consumed by both the single and list datasources. Slice fields
// are normalized to non-nil empty slices.
func FlattenPublicCloudVMTemplate(tpl *client.PublicCloudVMTemplate) map[string]interface{} {
	diskSizes := make([]interface{}, len(tpl.DiskSizesGb))
	for i, s := range tpl.DiskSizesGb {
		diskSizes[i] = s
	}
	families := make([]interface{}, len(tpl.CompatibleFamilies))
	for i, f := range tpl.CompatibleFamilies {
		families[i] = f
	}
	categories := make([]interface{}, len(tpl.Categories))
	for i, c := range tpl.Categories {
		categories[i] = c
	}
	return map[string]interface{}{
		"id":                  tpl.ID,
		"name":                tpl.Name,
		"os_family":           tpl.OsFamily,
		"os_name":             tpl.OsName,
		"os_version":          tpl.OsVersion,
		"disk_sizes_gb":       diskSizes,
		"compatible_families": families,
		"categories":          categories,
		"family":              tpl.Family,
		"version":             tpl.Version,
		"editor":              tpl.Editor,
		"description_en":      tpl.DescriptionEn,
		"template_type":       tpl.TemplateType,
		"icon":                tpl.Icon,
	}
}
