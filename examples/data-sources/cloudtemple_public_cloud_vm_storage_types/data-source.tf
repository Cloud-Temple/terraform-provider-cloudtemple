# Retrieve all storage types available to the tenant.
data "cloudtemple_public_cloud_vm_storage_types" "all" {}

output "storage_type_names" {
  value = [for s in data.cloudtemple_public_cloud_vm_storage_types.all.storage_types : s.name]
}
