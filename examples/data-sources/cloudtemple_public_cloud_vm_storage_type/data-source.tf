# Retrieve a storage type by name.
data "cloudtemple_public_cloud_vm_storage_type" "fast" {
  name = "Enterprise"
}

# The storage type id is consumed by the data-disk resource.
output "storage_type" {
  value = {
    id          = data.cloudtemple_public_cloud_vm_storage_type.fast.id
    iops_hint   = data.cloudtemple_public_cloud_vm_storage_type.fast.iops_hint
    max_size_gb = data.cloudtemple_public_cloud_vm_storage_type.fast.max_size_gb
  }
}
