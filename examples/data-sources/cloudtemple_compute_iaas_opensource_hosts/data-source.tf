# This example demonstrates how to use the data source to get the host details.
data "cloudtemple_compute_iaas_opensource_hosts" "hosts" {}

output "hosts" {
  value = data.cloudtemple_compute_iaas_opensource_hosts.hosts
}

data "cloudtemple_compute_iaas_opensource_hosts" "hosts_filtered" {
  machine_manager_id = "machine_manager_id"
  pool_id            = "pool_id"
}

output "hosts" {
  value = data.cloudtemple_compute_iaas_opensource_hosts.hosts
}