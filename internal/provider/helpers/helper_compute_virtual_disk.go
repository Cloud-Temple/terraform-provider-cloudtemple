package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenVirtualDisk convertit un objet VirtualDisk en une map compatible avec le schéma Terraform
func FlattenVirtualDisk(disk *client.VirtualDisk) map[string]interface{} {
	return map[string]interface{}{
		"name":                  disk.Name,
		"virtual_machine_id":    disk.VirtualMachineId,
		"machine_manager_id":    disk.MachineManager.ID,
		"capacity":              disk.Capacity,
		"disk_unit_number":      disk.DiskUnitNumber,
		"datastore_id":          disk.Datastore.ID,
		"datastore_name":        disk.Datastore.Name,
		"instant_access":        disk.InstantAccess,
		"native_id":             disk.NativeId,
		"disk_path":             disk.DiskPath,
		"provisioning_type":     disk.ProvisioningType,
		"disk_mode":             disk.DiskMode,
		"editable":              disk.Editable,
		"controller_id":         disk.Controller.ID,
		"controller_type":       disk.Controller.Type,
		"controller_bus_number": disk.Controller.BusNumber,
	}
}
