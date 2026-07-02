# Retrieve a region by name.
data "cloudtemple_public_cloud_vm_region" "fr1" {
  name = "fr1"
}

# Or by id.
data "cloudtemple_public_cloud_vm_region" "by_id" {
  id = "ed8bb3eb-a64c-46de-884f-65a5ea7d14c7"
}

output "region_country" {
  value = data.cloudtemple_public_cloud_vm_region.fr1.country_code
}

output "region_az_count" {
  value = data.cloudtemple_public_cloud_vm_region.fr1.az_count
}
