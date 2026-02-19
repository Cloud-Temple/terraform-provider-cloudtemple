package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSVirtualDisk convertit un objet OpenIaaSVirtualDisk en une map compatible avec le schéma Terraform
// vmID est l'ID de la VM configurée dans la ressource Terraform pour déterminer l'état de connexion
func FlattenOpenIaaSVirtualDisk(disk *client.OpenIaaSVirtualDisk, vmID string) map[string]interface{} {
	// Mapper les virtual machines
	virtualMachines := make([]map[string]interface{}, len(disk.VirtualMachines))
	for i, vm := range disk.VirtualMachines {
		virtualMachines[i] = map[string]interface{}{
			"id":        vm.ID,
			"name":      vm.Name,
			"read_only": vm.ReadOnly,
			"connected": vm.Connected,
		}
	}

	// Mapper les templates
	templates := make([]map[string]interface{}, len(disk.Templates))
	for i, template := range disk.Templates {
		templates[i] = map[string]interface{}{
			"id":        template.ID,
			"name":      template.Name,
			"read_only": template.ReadOnly,
		}
	}

	result := map[string]interface{}{
		"internal_id":           disk.InternalID,
		"name":                  disk.Name,
		"size":                  disk.Size,
		"usage":                 disk.Usage,
		"is_snapshot":           disk.IsSnapshot,
		"storage_repository_id": disk.StorageRepository.ID,
		"virtual_machines":      virtualMachines,
		"templates":             templates,
	}

	// Déterminer l'état de connexion pour la VM configurée
	if vmID != "" {
		for _, vm := range disk.VirtualMachines {
			if vm.ID == vmID {
				result["connected"] = vm.Connected
				break
			}
		}
	}

	return result
}
