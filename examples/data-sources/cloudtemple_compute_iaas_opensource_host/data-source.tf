# This example demonstrates how to use the data source to get the host details.
# The data source is used to fetch the host details based on the host ID or name and availability zone.

# Exemple with the ID of the host.
data "cloudtemple_compute_iaas_opensource_host" "host-1" {
  id = "host_id"
}

output "host-1" {
  value = data.cloudtemple_compute_iaas_opensource_host.host
}

# Exemple with the name of the host.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_compute_iaas_opensource_host" "host-2" {
  name               = "host_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "host-2" {
  value = data.cloudtemple_compute_iaas_opensource_host.host-2
}