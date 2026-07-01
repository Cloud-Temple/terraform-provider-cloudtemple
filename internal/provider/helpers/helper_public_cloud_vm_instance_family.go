package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMInstanceFamily maps a client.PublicCloudVMInstanceFamily to
// the flat snake_case map consumed by both the single and list datasources.
func FlattenPublicCloudVMInstanceFamily(family *client.PublicCloudVMInstanceFamily) map[string]interface{} {
	return map[string]interface{}{
		"id":          family.ID,
		"name":        family.Name,
		"description": family.Description,
		"vcpu_min":    family.VcpuMin,
		"vcpu_max":    family.VcpuMax,
		"ram_min_gb":  family.RamMinGb,
		"ram_max_gb":  family.RamMaxGb,
	}
}
