package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSStorageRepository convertit un objet OpenIaaSStorageRepository en une map compatible avec le schéma Terraform
func FlattenOpenIaaSStorageRepository(repository *client.OpenIaaSStorageRepository) map[string]interface{} {
	return map[string]interface{}{
		"id":                 repository.ID,
		"name":               repository.Name,
		"internal_id":        repository.InternalId,
		"description":        repository.Description,
		"maintenance_mode":   repository.MaintenanceMode,
		"max_capacity":       repository.MaxCapacity,
		"free_capacity":      repository.FreeCapacity,
		"type":               repository.StorageType,
		"virtual_disks":      repository.VirtualDisks,
		"shared":             repository.Shared,
		"accessible":         repository.Accessible,
		"host_id":            repository.Host.ID,
		"pool_id":            repository.Pool.ID,
		"machine_manager_id": repository.MachineManager.ID,
	}
}
