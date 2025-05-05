package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupOpenIaasBackup convertit un objet Backup en une map compatible avec le schéma Terraform
func FlattenBackupOpenIaasBackup(backup *client.Backup) map[string]interface{} {
	// Mapper la virtual machine
	virtualMachine := []map[string]interface{}{
		{
			"id":   backup.VirtualMachine.ID,
			"name": backup.VirtualMachine.Name,
		},
	}

	// Mapper la policy
	policy := []map[string]interface{}{
		{
			"id":   backup.Policy.ID,
			"name": backup.Policy.Name,
		},
	}

	return map[string]interface{}{
		"id":                         backup.ID,
		"internal_id":                backup.InternalID,
		"mode":                       backup.Mode,
		"virtual_machine":            virtualMachine,
		"policy":                     policy,
		"is_virtual_machine_deleted": backup.IsVirtualMachineDeleted,
		"size":                       backup.Size,
		"timestamp":                  backup.Timestamp,
	}
}
