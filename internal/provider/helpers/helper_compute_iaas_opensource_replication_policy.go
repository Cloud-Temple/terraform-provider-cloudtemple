package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSSnapshot convertit un objet OpenIaaSSnapshot en une map compatible avec le schéma Terraform
func FlattenOpenIaaSReplicationPolicy(policy *client.OpenIaaSReplicationPolicy) map[string]interface{} {
	return map[string]interface{}{
		"id":                    policy.ID,
		"name":                  policy.Name,
		"machine_manager_id":    policy.MachineManager.ID,
		"storage_repository_id": policy.StorageRepository.ID,
		"pool_id":               policy.Pool.ID,
		"interval": []interface{}{
			map[string]interface{}{
				"hours":   policy.Interval.Hours,
				"minutes": policy.Interval.Minutes,
			},
		},
		"last_run": []interface{}{
			map[string]interface{}{
				"start":  policy.LastRun.Start,
				"end":    policy.LastRun.End,
				"status": policy.LastRun.Status,
			},
		},
	}
}
