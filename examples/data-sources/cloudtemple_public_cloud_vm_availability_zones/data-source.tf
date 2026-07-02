# Retrieve all availability zones of the tenant.
data "cloudtemple_public_cloud_vm_availability_zones" "all" {}

output "zone_names" {
  value = [for z in data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones : z.name]
}

# Filter in HCL, e.g. the zones of a given region.
output "fr1_zones" {
  value = [
    for z in data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones :
    z.name if startswith(z.name, "fr1-")
  ]
}
