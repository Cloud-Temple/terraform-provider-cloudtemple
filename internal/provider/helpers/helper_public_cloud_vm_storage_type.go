package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMStorageType maps a client.PublicCloudVMStorageType to the
// flat snake_case map consumed by both the single and list datasources. A nil
// Sku (the API may omit it or send null) flattens to an empty list, mirroring
// the nested-single-object idiom used across the Public Cloud VM datasources.
func FlattenPublicCloudVMStorageType(st *client.PublicCloudVMStorageType) map[string]interface{} {
	sku := []map[string]interface{}{}
	if st.Sku != nil {
		sku = []map[string]interface{}{{
			"name":           st.Sku.Name,
			"price":          st.Sku.Price,
			"unit":           st.Sku.Unit,
			"description":    st.Sku.Description,
			"description_en": st.Sku.DescriptionEn,
		}}
	}
	return map[string]interface{}{
		"id":           st.ID,
		"name":         st.Name,
		"description":  st.Description,
		"iops_hint":    st.IopsHint,
		"min_size_gb":  st.MinSizeGb,
		"max_size_gb":  st.MaxSizeGb,
		"is_available": st.IsAvailable,
		"sku":          sku,
	}
}
