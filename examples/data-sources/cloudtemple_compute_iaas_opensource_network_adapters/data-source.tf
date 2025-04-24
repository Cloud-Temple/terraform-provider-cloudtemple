# This example demonstrates how to use the data source to get network adapters from an Open IaaS infrastructure.
# The data source is used to fetch all network adapters for a specific virtual machine.

# Basic example - Retrieve all network adapters for a specific virtual machine
data "cloudtemple_compute_iaas_opensource_network_adapters" "all" {
  virtual_machine_id = "virtual_machine_id"
}

output "all_network_adapters" {
  value = data.cloudtemple_compute_iaas_opensource_network_adapters.all.network_adapters
}