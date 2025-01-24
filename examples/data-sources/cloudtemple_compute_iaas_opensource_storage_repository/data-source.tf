# This example demonstrates how to use the data source to get the storage repository details.
# The data source is used to fetch the storage repository details based on the storage repository ID or name and availability zone.

# Example with the ID of the storage repository.
data "cloudtemple_compute_iaas_opensource_storage_repository" "storage_repository-1" {
  id = "storage_repository_id"
}

output "storage_repository-1" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repository.storage_repository-1
}

# Exemple with the name of the storage repository.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

# See the available values for the type attribute at the following URL:
# https://registry.terraform.io/providers/Cloud-Temple/cloudtemple/latest/docs/data-sources/compute_iaas_opensource_storage_repository#type-1
data "cloudtemple_compute_iaas_opensource_storage_repository" "storage_repository-2" {
  name               = "sr_name"
  type               = "lvmohba"
  shared             = true
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "storage_repository-2" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repository.storage_repository-2
}