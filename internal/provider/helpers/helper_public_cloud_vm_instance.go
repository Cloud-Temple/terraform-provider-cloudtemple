package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// flattenPublicCloudVMInstanceRef maps a {id, name} reference to the single-element
// list shape used by the datasource nested blocks.
func flattenPublicCloudVMInstanceRef(r client.PublicCloudVMInstanceRef) []map[string]interface{} {
	return []map[string]interface{}{{"id": r.ID, "name": r.Name}}
}

// FlattenPublicCloudVMInstance maps a client.PublicCloudVMInstance to the flat
// snake_case map consumed by both the single and list datasources. A nil
// BackupPolicy (the API returns it null) flattens to an empty list.
func FlattenPublicCloudVMInstance(vm *client.PublicCloudVMInstance) map[string]interface{} {
	backupPolicy := []map[string]interface{}{}
	if vm.BackupPolicy != nil {
		backupPolicy = flattenPublicCloudVMInstanceRef(*vm.BackupPolicy)
	}
	return map[string]interface{}{
		"id":                    vm.ID,
		"name":                  vm.Name,
		"status":                vm.Status,
		"availability_zone":     flattenPublicCloudVMInstanceRef(vm.AZ),
		"template":              flattenPublicCloudVMInstanceRef(vm.Template),
		"instance_family":       flattenPublicCloudVMInstanceRef(vm.InstanceFamily),
		"vcpu":                  vm.VCPU,
		"ram_gb":                vm.RAMGb,
		"disks_size_gb":         vm.DisksSizeGb,
		"backup_policy":         backupPolicy,
		"guest_tools_installed": vm.GuestToolsInstalled,
		"created_at":            vm.CreatedAt,
		"updated_at":            vm.UpdatedAt,
	}
}
