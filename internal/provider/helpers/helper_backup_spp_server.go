package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupSPPServer convertit un objet BackupSPPServer en une map compatible avec le schéma Terraform
func FlattenBackupSPPServer(server *client.BackupSPPServer) map[string]interface{} {
	return map[string]interface{}{
		"id":      server.ID,
		"name":    server.Name,
		"address": server.Address,
	}
}
