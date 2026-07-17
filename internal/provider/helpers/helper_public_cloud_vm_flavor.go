package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMFlavor maps a client.PublicCloudVMFlavor to the flat
// snake_case map consumed by both the single and list datasources.
func FlattenPublicCloudVMFlavor(flavor *client.PublicCloudVMFlavor) map[string]interface{} {
	return map[string]interface{}{
		"id":                 flavor.ID,
		"instance_family_id": flavor.InstanceFamilyID,
		"name":               flavor.Name,
		"vcpu":               flavor.Vcpu,
		"ram_gb":             flavor.RamGb,
	}
}
