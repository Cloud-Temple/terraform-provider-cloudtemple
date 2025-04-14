package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSTemplate convertit un objet OpenIaasTemplate en une map compatible avec le schéma Terraform
func FlattenOpenIaaSTemplate(template *client.OpenIaasTemplate) map[string]interface{} {
	return map[string]interface{}{
		"id":                   template.ID,
		"name":                 template.Name,
		"internal_id":          template.InternalID,
		"cpu":                  template.CPU,
		"num_cores_per_socket": template.NumCoresPerSocket,
		"memory":               template.Memory,
		"power_state":          template.PowerState,
		"snapshots":            template.Snapshots,
		"sla_policies":         template.SLAPolicies,
		"disks":                flattenTemplateDisksList(template.Disks),
		"network_adapters":     flattenTemplateNetworkAdaptersList(template.NetworkAdapters),
		"machine_manager_id":   template.MachineManager.ID,
	}
}

// flattenTemplateDisksList convertit une liste de TemplateDisk en une liste compatible avec le schéma Terraform
func flattenTemplateDisksList(disks []client.TemplateDisk) []interface{} {
	if disks == nil {
		return make([]interface{}, 0)
	}

	result := make([]interface{}, len(disks))
	for i, disk := range disks {
		result[i] = map[string]interface{}{
			"name":               disk.Name,
			"description":        disk.Description,
			"size":               disk.Size,
			"storage_repository": flattenBaseObjectList(disk.StorageRepository),
		}
	}
	return result
}

// flattenTemplateNetworkAdaptersList convertit une liste de TemplateNetworkAdapter en une liste compatible avec le schéma Terraform
func flattenTemplateNetworkAdaptersList(adapters []client.TemplateNetworkAdapter) []interface{} {
	if adapters == nil {
		return make([]interface{}, 0)
	}

	result := make([]interface{}, len(adapters))
	for i, adapter := range adapters {
		result[i] = map[string]interface{}{
			"name":        adapter.Name,
			"mac_address": adapter.MacAddress,
			"mtu":         adapter.MTU,
			"attached":    adapter.Attached,
			"network":     flattenBaseObjectList(adapter.Network),
		}
	}
	return result
}

// flattenBaseObjectList convertit un BaseObject en une liste compatible avec le schéma Terraform
func flattenBaseObjectList(obj client.BaseObject) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"id":   obj.ID,
			"name": obj.Name,
		},
	}
}
