# This example demonstrates how to use the data source to get pools from an Open IaaS infrastructure.
# The data source is used to fetch all pools or filter them by machine manager ID.

# Basic example - Retrieve all pools
data "cloudtemple_compute_iaas_opensource_pools" "all" {
}

output "all_pools" {
  value = data.cloudtemple_compute_iaas_opensource_pools.all.pools
}

# Example with machine_manager_id filter
data "cloudtemple_compute_iaas_opensource_pools" "by_machine_manager" {
  machine_manager_id = "machine_manager_id"
}

output "pools_by_machine_manager" {
  value = data.cloudtemple_compute_iaas_opensource_pools.by_machine_manager.pools
}
