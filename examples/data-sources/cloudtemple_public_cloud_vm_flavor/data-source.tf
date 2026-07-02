# Retrieve a flavor (a predefined vCPU/RAM sizing pair) by name.
data "cloudtemple_public_cloud_vm_flavor" "medium" {
  name = "m2.medium"
}

output "flavor_sizing" {
  value = {
    vcpu   = data.cloudtemple_public_cloud_vm_flavor.medium.vcpu
    ram_gb = data.cloudtemple_public_cloud_vm_flavor.medium.ram_gb
  }
}
