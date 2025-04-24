# This example demonstrates how to use the data source to get virtual machines from an Open IaaS infrastructure.
# The data source is used to fetch all virtual machines or filter them by machine manager ID.

# Basic example - Retrieve all virtual machines
data "cloudtemple_compute_iaas_opensource_virtual_machines" "all" {
}

output "all_virtual_machines" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_machines.all.virtual_machines
}

# Example with reference to another data source
# First, get an availability zone
data "cloudtemple_compute_iaas_opensource_availability_zone" "example" {
  name = "example_zone_name"
}

# Then retrieve all virtual machines for that machine manager
data "cloudtemple_compute_iaas_opensource_virtual_machines" "for_machine_manager" {
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.example.id
}

# Output the virtual machines
output "virtual_machines_for_machine_manager" {
  value = data.cloudtemple_compute_iaas_opensource_virtual_machines.for_machine_manager.virtual_machines
}