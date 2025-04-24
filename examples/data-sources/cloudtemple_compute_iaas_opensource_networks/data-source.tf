# This example demonstrates how to use the data source to get networks from an Open IaaS infrastructure.
# The data source is used to fetch all networks or filter them by machine manager ID or pool ID.

# Basic example - Retrieve all networks
data "cloudtemple_compute_iaas_opensource_networks" "all" {
}

output "all_networks" {
  value = data.cloudtemple_compute_iaas_opensource_networks.all.networks
}

# Example with machine_manager_id filter
data "cloudtemple_compute_iaas_opensource_networks" "by_machine_manager" {
  machine_manager_id = "machine_manager_id"
}

output "networks_by_machine_manager" {
  value = data.cloudtemple_compute_iaas_opensource_networks.by_machine_manager.networks
}

# Example with pool_id filter
data "cloudtemple_compute_iaas_opensource_networks" "by_pool" {
  pool_id = "pool_id"
}

output "networks_by_pool" {
  value = data.cloudtemple_compute_iaas_opensource_networks.by_pool.networks
}