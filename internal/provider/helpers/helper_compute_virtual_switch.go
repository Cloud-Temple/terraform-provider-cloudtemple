package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenVirtualSwitch convertit un objet VirtualSwitch en une map compatible avec le schéma Terraform
func FlattenVirtualSwitch(virtualSwitch *client.VirtualSwitch) map[string]interface{} {
	return map[string]interface{}{
		"id":                 virtualSwitch.ID,
		"name":               virtualSwitch.Name,
		"moref":              virtualSwitch.Moref,
		"folder_id":          virtualSwitch.FolderID,
		"machine_manager_id": virtualSwitch.MachineManager.ID,
	}
}
