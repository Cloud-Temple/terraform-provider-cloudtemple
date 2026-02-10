package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenDatastore convertit un objet Datastore en une map compatible avec le schéma Terraform
func FlattenDatastore(datastore *client.Datastore) map[string]interface{} {
	return map[string]interface{}{
		"id":                      datastore.ID,
		"name":                    datastore.Name,
		"machine_manager_id":      datastore.MachineManager.ID,
		"moref":                   datastore.Moref,
		"max_capacity":            datastore.MaxCapacity,
		"free_capacity":           datastore.FreeCapacity,
		"accessible":              datastore.Accessible,
		"maintenance_status":      datastore.MaintenanceMode, // Deprecated: use maintenance_mode
		"maintenance_mode":        datastore.MaintenanceMode,
		"unique_id":               datastore.UniqueId,
		"type":                    datastore.Type,
		"virtual_machines_number": datastore.VirtualMachinesNumber,
		"hosts_number":            datastore.HostsNumber,
		"hosts_names":             datastore.HostsNames,
		"associated_folder":       datastore.AssociatedFolder,
	}
}
