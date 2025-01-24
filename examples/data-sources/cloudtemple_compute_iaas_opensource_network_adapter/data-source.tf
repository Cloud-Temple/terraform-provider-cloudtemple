# This example demonstrates how to use the data source to get the network adapter details.
# The data source is used to fetch the network adapter details based on the network adapter ID or name and virtual machine

# Example with the ID of the network adapter.
data "cloudtemple_compute_iaas_opensource_network_adapter" "network_adapter-1" {
  id = "network_adapter_id"
}

output "network_adapter-1" {
  value = data.cloudtemple_compute_iaas_opensource_network_adapter.network_adapter-1
}

# Exemple with the name of the network adapter.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_compute_iaas_opensource_network_adapter" "network_adapter-2" {
  name               = "network_adapter_name"
  virtual_machine_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "network_adapter-2" {
  value = data.cloudtemple_compute_iaas_opensource_network_adapter.network_adapter-2
}