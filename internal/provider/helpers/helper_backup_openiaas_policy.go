package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupOpenIaasPolicy convertit un objet BackupOpenIaasPolicy en une map compatible avec le schéma Terraform
func FlattenBackupOpenIaasPolicy(policy *client.BackupOpenIaasPolicy) map[string]interface{} {
	// Mapper les schedulers
	schedulers := make([]map[string]interface{}, len(policy.Schedulers))
	for i, scheduler := range policy.Schedulers {
		schedulers[i] = map[string]interface{}{
			"temporarily_disabled": scheduler.TemporarilyDisabled,
			"retention":            scheduler.Retention,
			"cron":                 scheduler.Cron,
			"timezone":             scheduler.Timezone,
		}
	}

	return map[string]interface{}{
		"id":                   policy.ID,
		"name":                 policy.Name,
		"internal_id":          policy.InternalID,
		"running":              policy.Running,
		"mode":                 policy.Mode,
		"machine_manager_id":   policy.MachineManager.ID,
		"machine_manager_name": policy.MachineManager.Name,
		"schedulers":           schedulers,
	}
}
