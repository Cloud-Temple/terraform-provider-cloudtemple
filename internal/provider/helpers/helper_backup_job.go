package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupJob convertit un objet BackupJob en une map compatible avec le schéma Terraform
func FlattenBackupJob(job *client.BackupJob) map[string]interface{} {
	return map[string]interface{}{
		"id":           job.ID,
		"name":         job.Name,
		"display_name": job.DisplayName,
		"type":         job.Type,
		"status":       job.Status,
		"policy_id":    job.PolicyId,
	}
}
