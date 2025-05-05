# This example demonstrates how to use the data source to get the availability zone details.
# The data source is used to fetch the availability zone details based on the availability zone ID or name.

# Exemple with the ID of the availability zone.
data "cloudtemple_compute_iaas_opensource_availability_zones" "availability_zones" {}

output "availability_zones" {
  value = data.cloudtemple_compute_iaas_opensource_availability_zones.availability_zones
}