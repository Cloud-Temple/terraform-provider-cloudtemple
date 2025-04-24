package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSSnapshot convertit un objet OpenIaaSSnapshot en une map compatible avec le schéma Terraform
func FlattenOpenIaaSSnapshot(snapshot *client.OpenIaaSSnapshot) map[string]interface{} {
	return map[string]interface{}{
		"id":                 snapshot.ID,
		"name":               snapshot.Name,
		"description":        snapshot.Description,
		"virtual_machine_id": snapshot.VirtualMachineID,
		"create_time":        snapshot.CreateTime,
	}
}
