package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenVirtualDatacenter convertit un objet VirtualDatacenter en une map compatible avec le schéma Terraform
func FlattenVirtualDatacenter(datacenter *client.VirtualDatacenter) map[string]interface{} {
	// Mapper le vcenter
	vcenter := []map[string]interface{}{
		{
			"id":   datacenter.VCenter.ID,
			"name": datacenter.VCenter.Name,
		},
	}

	return map[string]interface{}{
		"id":                 datacenter.ID,
		"name":               datacenter.Name,
		"tenant_id":          datacenter.TenantID,
		"vcenter":            vcenter,
		"machine_manager_id": datacenter.MachineManager.ID,
	}
}
