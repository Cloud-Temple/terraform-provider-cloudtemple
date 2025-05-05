package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenVirtualDatacenter convertit un objet VirtualDatacenter en une map compatible avec le schéma Terraform
func FlattenVirtualDatacenter(datacenter *client.VirtualDatacenter) map[string]interface{} {
	return map[string]interface{}{
		"id":                 datacenter.ID,
		"name":               datacenter.Name,
		"tenant_id":          datacenter.TenantID,
		"machine_manager_id": datacenter.MachineManager.ID,
	}
}
