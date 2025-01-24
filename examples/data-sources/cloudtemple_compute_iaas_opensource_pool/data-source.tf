# This example demonstrates how to use the data source to get the pool details.
# The data source is used to fetch the pool details based on the pool ID or name and availability zone.

# Example with the ID of the pool.
data "cloudtemple_compute_iaas_opensource_pool" "pool-1" {
  id = "pool_id"
}

output "pool-1" {
  value = data.cloudtemple_compute_iaas_opensource_pool.pool-1
}

# Exemple with the name of the pool.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_compute_iaas_opensource_pool" "pool-2" {
  name               = "pool_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "pool-2" {
  value = data.cloudtemple_compute_iaas_opensource_pool.pool-2
}