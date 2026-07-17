package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMStorageType maps a client.PublicCloudVMStorageType to the
// flat snake_case map consumed by both the single and list datasources.
func FlattenPublicCloudVMStorageType(st *client.PublicCloudVMStorageType) map[string]interface{} {
	return map[string]interface{}{
		"id":           st.ID,
		"name":         st.Name,
		"description":  st.Description,
		"iops_hint":    st.IopsHint,
		"min_size_gb":  st.MinSizeGb,
		"max_size_gb":  st.MaxSizeGb,
		"is_available": st.IsAvailable,
	}
}
