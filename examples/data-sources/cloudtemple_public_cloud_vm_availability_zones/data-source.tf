# Retrieve all availability zones of the tenant.
data "cloudtemple_public_cloud_vm_availability_zones" "all" {}

# Or only those of a given region.
data "cloudtemple_public_cloud_vm_availability_zones" "fr1" {
  region_id = "ed8bb3eb-a64c-46de-884f-65a5ea7d14c7"
}

output "az_names" {
  value = [for z in data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones : z.name]
}
