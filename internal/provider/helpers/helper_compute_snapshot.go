package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenSnapshot convertit un objet Snapshot en une map compatible avec le schéma Terraform
func FlattenSnapshot(snapshot *client.Snapshot) map[string]interface{} {
	return map[string]interface{}{
		"id":                 snapshot.ID,
		"name":               snapshot.Name,
		"virtual_machine_id": snapshot.VirtualMachineId,
		"create_time":        snapshot.CreateTime,
	}
}
