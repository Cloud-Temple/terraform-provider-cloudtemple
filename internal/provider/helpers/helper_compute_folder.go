package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenFolder convertit un objet Folder en une map compatible avec le schéma Terraform
func FlattenFolder(folder *client.Folder) map[string]interface{} {
	return map[string]interface{}{
		"id":                 folder.ID,
		"name":               folder.Name,
		"machine_manager_id": folder.MachineManager.ID,
	}
}
