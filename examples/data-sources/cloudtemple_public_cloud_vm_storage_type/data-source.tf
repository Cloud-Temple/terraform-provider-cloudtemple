# Retrieve a Public Cloud VM Instances storage type by name.
data "cloudtemple_public_cloud_vm_storage_type" "standard" {
  name = "Standard"
}

output "standard_max_size_gb" {
  value = data.cloudtemple_public_cloud_vm_storage_type.standard.max_size_gb
}
