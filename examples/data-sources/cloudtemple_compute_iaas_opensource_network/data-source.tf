# This example demonstrates how to use the data source to get the network details.
# The data source is used to fetch the network details based on the network ID or name and availability zone.

# Example with the ID of the network.
data "cloudtemple_compute_iaas_opensource_network" "network-1" {
  id = "network_id"
}

output "network-1" {
  value = data.cloudtemple_compute_iaas_opensource_network.network-1
}

# Exemple with the name of the network.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_compute_iaas_opensource_network" "network-2" {
  name               = "network_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "network-2" {
  value = data.cloudtemple_compute_iaas_opensource_network.network-2
}