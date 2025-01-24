# This example demonstrates how to use the data source to get the virtual machine details.
# The data source is used to fetch the virtual machine details based on the virtual machine ID or name and availability zone.

# Example with the ID of the virtual machine.
data "cloudtemple_compute_iaas_opensource_virtual_machine" "virtual_machine-1" {
  id = "virtual_machine_id"
}

output "virtual_machine-1" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_machine.virtual_machine-1
}

# Exemple with the name of the virtual machine.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_compute_iaas_opensource_virtual_machine" "virtual_machine-2" {
  name               = "virtual_machine_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "virtual_machine-2" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_machine.virtual_machine-2
}