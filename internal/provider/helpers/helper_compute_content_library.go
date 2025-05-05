package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenContentLibrary convertit un objet ContentLibrary en une map compatible avec le schéma Terraform
func FlattenContentLibrary(contentLibrary *client.ContentLibrary) map[string]interface{} {
	// Mapper le datastore
	datastore := []map[string]interface{}{
		{
			"id":   contentLibrary.Datastore.ID,
			"name": contentLibrary.Datastore.Name,
		},
	}

	return map[string]interface{}{
		"id":                 contentLibrary.ID,
		"name":               contentLibrary.Name,
		"machine_manager_id": contentLibrary.MachineManager.ID,
		"type":               contentLibrary.Type,
		"datastore":          datastore,
	}
}

// FlattenContentLibraryItem convertit un objet ContentLibraryItem en une map compatible avec le schéma Terraform
func FlattenContentLibraryItem(item *client.ContentLibraryItem) map[string]interface{} {
	return map[string]interface{}{
		"id":                 item.ID,
		"name":               item.Name,
		"description":        item.Description,
		"type":               item.Type,
		"creation_time":      item.CreationTime.Format("2006-01-02T15:04:05Z07:00"),
		"size":               item.Size,
		"stored":             item.Stored,
		"last_modified_time": item.LastModifiedTime,
		"ovf_properties":     item.OvfProperties,
	}
}
