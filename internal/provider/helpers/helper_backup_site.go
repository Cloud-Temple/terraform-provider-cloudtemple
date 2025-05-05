package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupSite convertit un objet BackupSite en une map compatible avec le schéma Terraform
func FlattenBackupSite(site *client.BackupSite) map[string]interface{} {
	return map[string]interface{}{
		"id":   site.ID,
		"name": site.Name,
	}
}
