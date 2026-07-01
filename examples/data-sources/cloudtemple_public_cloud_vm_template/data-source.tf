# Retrieve a Public Cloud VM Instances template (OS image) by name.
data "cloudtemple_public_cloud_vm_template" "rocky9" {
  name = "Rocky Linux 9"
}

output "rocky9_disk_sizes" {
  value = data.cloudtemple_public_cloud_vm_template.rocky9.disk_sizes_gb
}
