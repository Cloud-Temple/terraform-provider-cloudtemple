# Retrieve the full network catalogue of the tenant. There is no server-side
# filter — filter in HCL.
data "cloudtemple_public_cloud_vm_networks" "all" {}

output "network_names" {
  value = [for n in data.cloudtemple_public_cloud_vm_networks.all.networks : n.name]
}

# A non-empty `vpc` block identifies a VPC network.
output "vpc_networks" {
  value = [
    for n in data.cloudtemple_public_cloud_vm_networks.all.networks :
    n.name if length(n.vpc) > 0
  ]
}
