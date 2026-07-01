# Retrieve the full Public Cloud VM Instances network catalogue of the tenant.
data "cloudtemple_public_cloud_vm_networks" "all" {}

output "network_names" {
  value = [for n in data.cloudtemple_public_cloud_vm_networks.all.networks : n.name]
}
