package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSMachineManager convertit un objet OpenIaaSMachineManager en une map compatible avec le schéma Terraform
func FlattenOpenIaaSMachineManager(machineManager *client.OpenIaaSMachineManager) map[string]interface{} {
	return map[string]interface{}{
		"id":          machineManager.ID,
		"name":        machineManager.Name,
		"os_version":  machineManager.OSVersion,
		"os_name":     machineManager.OSName,
		"xoa_version": machineManager.XOAVersion,
	}
}
