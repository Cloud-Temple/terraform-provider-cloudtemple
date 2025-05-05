package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupJobSession convertit un objet BackupJobSession en une map compatible avec le schéma Terraform
func FlattenBackupJobSession(js *client.BackupJobSession) map[string]interface{} {
	// Mapper les statistiques
	statistics := map[string]interface{}{
		"total":   js.Statistics.Total,
		"success": js.Statistics.Success,
		"failed":  js.Statistics.Failed,
		"skipped": js.Statistics.Skipped,
	}

	// Mapper les SLA policies
	slaPolicies := make([]map[string]interface{}, len(js.SLAPolicies))
	for j, policy := range js.SLAPolicies {
		slaPolicies[j] = map[string]interface{}{
			"id":   policy.ID,
			"name": policy.Name,
			"href": policy.HREF,
		}
	}

	// Créer l'entrée pour cette session
	return map[string]interface{}{
		"id":              js.ID,
		"job_name":        js.JobName,
		"sla_policy_type": js.SlaPolicyType,
		"job_id":          js.JobId,
		"type":            js.Type,
		"duration":        js.Duration,
		"start":           js.Start,
		"end":             js.End,
		"status":          js.Status,
		"statistics":      []interface{}{statistics},
		"sla_policies":    slaPolicies,
	}
}
