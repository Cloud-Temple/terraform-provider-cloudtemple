# This example demonstrates how to use the data source to get the template details.
# The data source is used to fetch the template details based on the template ID or name and availability zone.

# Example with the ID of the template.
data "cloudtemple_compute_iaas_opensource_template" "template-1" {
  id = "template_id"
}

output "template-1" {
  value = data.cloudtemple_compute_iaas_opensource_template.template-1
}

# Exemple with the name of the template.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_compute_iaas_opensource_template" "template-2" {
  name               = "template_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "template-2" {
  value = data.cloudtemple_compute_iaas_opensource_template.template-2
}