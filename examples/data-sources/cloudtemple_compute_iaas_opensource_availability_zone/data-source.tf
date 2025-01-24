# This example demonstrates how to use the data source to get the availability zone details.
# The data source is used to fetch the availability zone details based on the availability zone ID or name.

# Exemple with the ID of the availability zone.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone-1" {
  id = "availability_zone_id"
}

output "availability_zone-1" {
  value = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone-1
}

# Exemple with the name of the availability zone.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone-2" {
  name = "availability_zone_name"
}

output "availability_zone-2" {
  value = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone-2
}