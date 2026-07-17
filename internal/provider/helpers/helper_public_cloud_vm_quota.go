package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMQuota maps a client.PublicCloudVMQuota to the flat
// snake_case map consumed by the quota datasource.
func FlattenPublicCloudVMQuota(quota *client.PublicCloudVMQuota) map[string]interface{} {
	return map[string]interface{}{
		"vcpu_limit":       quota.VcpuLimit,
		"ram_limit_mb":     quota.RamLimitMb,
		"storage_limit_gb": quota.StorageLimitGb,
		"vcpu_used":        quota.VcpuUsed,
		"ram_used_mb":      quota.RamUsedMb,
		"storage_used_gb":  quota.StorageUsedGb,
	}
}
