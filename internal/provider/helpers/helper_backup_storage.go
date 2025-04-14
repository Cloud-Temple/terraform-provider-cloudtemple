package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupStorage convertit un objet BackupStorage en une map compatible avec le schéma Terraform
func FlattenBackupStorage(storage *client.BackupStorage) map[string]interface{} {
	// Mapper la capacité
	capacity := []map[string]interface{}{
		{
			"free":        storage.Capacity.Free,
			"total":       storage.Capacity.Total,
			"update_time": storage.Capacity.UpdateTime,
		},
	}

	return map[string]interface{}{
		"id":                storage.ID,
		"resource_type":     storage.ResourceType,
		"type":              storage.Type,
		"site":              storage.Site,
		"name":              storage.Name,
		"storage_id":        storage.StorageId,
		"host_address":      storage.HostAddress,
		"port_number":       storage.PortNumber,
		"ssl_connection":    storage.SSLConnection,
		"initialize_status": storage.InitializeStatus,
		"version":           storage.Version,
		"is_ready":          storage.IsReady,
		"capacity":          capacity,
	}
}
