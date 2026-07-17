# Retrieve all storage types available to the tenant.
data "cloudtemple_public_cloud_vm_storage_types" "all" {}

output "available_storage_types" {
  value = [
    for st in data.cloudtemple_public_cloud_vm_storage_types.all.storage_types :
    "${st.name} (${st.iops_hint})" if st.is_available
  ]
}
