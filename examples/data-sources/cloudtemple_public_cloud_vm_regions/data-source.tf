# Retrieve all Public Cloud VM Instances regions of the tenant.
data "cloudtemple_public_cloud_vm_regions" "all" {}

output "region_names" {
  value = [for r in data.cloudtemple_public_cloud_vm_regions.all.regions : r.name]
}
