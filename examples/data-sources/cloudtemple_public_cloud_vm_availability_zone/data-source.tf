# Retrieve an availability zone by name.
data "cloudtemple_public_cloud_vm_availability_zone" "az1" {
  name = "fr1-az01"
}

# Or by id.
data "cloudtemple_public_cloud_vm_availability_zone" "by_id" {
  id = "18eea4f6-b4a1-40c1-8435-457ee7a9ab8e"
}

# The zone id is what a VM instance deploys into.
output "az_id" {
  value = data.cloudtemple_public_cloud_vm_availability_zone.az1.id
}

output "az_compatible_families" {
  value = data.cloudtemple_public_cloud_vm_availability_zone.az1.compatible_families
}
