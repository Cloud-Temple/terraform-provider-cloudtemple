# Retrieve a Public Cloud VM Instances availability zone by name.
data "cloudtemple_public_cloud_vm_availability_zone" "az01" {
  name = "fr1-az01"
}

# Or by id.
data "cloudtemple_public_cloud_vm_availability_zone" "by_id" {
  id = "18eea4f6-b4a1-40c1-8435-457ee7a9ab8e"
}

output "az_region" {
  value = data.cloudtemple_public_cloud_vm_availability_zone.az01.region_id
}
