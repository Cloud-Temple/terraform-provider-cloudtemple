package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupVCenter convertit un objet BackupVCenter en une map compatible avec le schéma Terraform
func FlattenBackupVCenter(vcenter *client.BackupVCenter) map[string]interface{} {
	return map[string]interface{}{
		"id":            vcenter.ID,
		"internal_id":   vcenter.InternalId,
		"instance_id":   vcenter.InstanceId,
		"spp_server_id": vcenter.SppServerId,
		"name":          vcenter.Name,
	}
}
