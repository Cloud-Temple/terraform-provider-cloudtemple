# This example demonstrates how to use the data source to get the virtual disk details.
# The data source is used to fetch the virtual disk details based on the virtual disk ID or name and availability zone.

# Example with the ID of the virtual disk.
data "cloudtemple_compute_iaas_opensource_virtual_disk" "virtual_disk-1" {
  id = "virtual_disk_id"
}

output "virtual_disk-1" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_disk.virtual_disk-1
}

# Exemple with the name of the virtual disk.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_compute_iaas_opensource_virtual_disk" "virtual_disk-2" {
  name               = "virtual_disk_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "virtual_disk-2" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_disk.virtual_disk-2
}