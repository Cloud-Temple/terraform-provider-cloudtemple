package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenBackupSLAPolicy convertit un objet BackupSLAPolicy en une map compatible avec le schéma Terraform
func FlattenBackupSLAPolicy(policy *client.BackupSLAPolicy) map[string]interface{} {
	// Mapper les sous-politiques
	subPolicies := make([]map[string]interface{}, len(policy.SubPolicies))
	for i, subPolicy := range policy.SubPolicies {
		// Mapper la rétention
		retention := []map[string]interface{}{
			{
				"age": subPolicy.Retention.Age,
			},
		}

		// Mapper le trigger
		trigger := []map[string]interface{}{
			{
				"frequency":     subPolicy.Trigger.Frequency,
				"type":          subPolicy.Trigger.Type,
				"activate_date": subPolicy.Trigger.ActivateDate,
			},
		}

		// Mapper la cible
		target := []map[string]interface{}{
			{
				"id":            subPolicy.Target.ID,
				"href":          subPolicy.Target.Href,
				"resource_type": subPolicy.Target.ResourceType,
			},
		}

		subPolicies[i] = map[string]interface{}{
			"type":           subPolicy.Type,
			"use_encryption": subPolicy.UseEncryption,
			"software":       subPolicy.Software,
			"site":           subPolicy.Site,
			"retention":      retention,
			"trigger":        trigger,
			"target":         target,
		}
	}

	return map[string]interface{}{
		"id":           policy.ID,
		"name":         policy.Name,
		"sub_policies": subPolicies,
	}
}
