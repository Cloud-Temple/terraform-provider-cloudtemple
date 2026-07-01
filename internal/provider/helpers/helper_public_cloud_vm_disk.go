package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMDisk maps a client.PublicCloudVMDisk to the flat snake_case
// map consumed by the disks datasource.
func FlattenPublicCloudVMDisk(d *client.PublicCloudVMDisk) map[string]interface{} {
	return map[string]interface{}{
		"id":           d.ID,
		"position":     d.Position,
		"name":         d.Label,
		"size_gb":      d.SizeGb,
		"storage_type": d.StorageType,
		"is_primary":   d.IsPrimary,
	}
}
