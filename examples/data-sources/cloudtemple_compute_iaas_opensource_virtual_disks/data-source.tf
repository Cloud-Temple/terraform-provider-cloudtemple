# This example demonstrates how to use the data source to get virtual disks from an Open IaaS infrastructure.
# The data source is used to fetch all virtual disks or filter them by various criteria.

# Basic example - Retrieve all virtual disks
data "cloudtemple_compute_iaas_opensource_virtual_disks" "all" {
}

output "all_virtual_disks" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_disks.all.virtual_disks
}

# Example with virtual_machine_id filter
data "cloudtemple_compute_iaas_opensource_virtual_disks" "by_vm" {
  virtual_machine_id = "virtual_machine_id"
}

output "virtual_disks_by_vm" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_disks.by_vm.virtual_disks
}

# Example with template_id filter
data "cloudtemple_compute_iaas_opensource_virtual_disks" "by_template" {
  template_id = "template_id"
}

output "virtual_disks_by_template" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_disks.by_template.virtual_disks
}

# Example with storage_repository_id filter
data "cloudtemple_compute_iaas_opensource_virtual_disks" "by_storage_repository" {
  storage_repository_id = "storage_repository_id"
}

output "virtual_disks_by_storage_repository" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_disks.by_storage_repository.virtual_disks
}
