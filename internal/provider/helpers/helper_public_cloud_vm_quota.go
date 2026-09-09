package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMQuota maps a client.PublicCloudVMQuota to the flat
// snake_case map consumed by the quota datasource.
func FlattenPublicCloudVMQuota(quota *client.PublicCloudVMQuota) map[string]interface{} {
	return map[string]interface{}{
		"vcpu_limit":        quota.VcpuLimit,
		"ram_limit_mib":     quota.RamLimitMib,
		"storage_limit_gib": quota.StorageLimitGib,
		"vcpu_used":         quota.VcpuUsed,
		"ram_used_mib":      quota.RamUsedMib,
		"storage_used_gib":  quota.StorageUsedGib,
	}
}
