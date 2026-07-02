# Retrieve all regions of the tenant.
data "cloudtemple_public_cloud_vm_regions" "all" {}

output "region_names" {
  value = [for r in data.cloudtemple_public_cloud_vm_regions.all.regions : r.name]
}

# Only the enabled regions.
output "enabled_regions" {
  value = [for r in data.cloudtemple_public_cloud_vm_regions.all.regions : r.name if r.is_enabled]
}
