# Retrieve an instance family by name.
data "cloudtemple_public_cloud_vm_instance_family" "family" {
  name = "Development"
}

# The family id is required to create a VM, and bounds its cpu/memory sizing.
output "family_sizing_bounds" {
  value = {
    vcpu_min   = data.cloudtemple_public_cloud_vm_instance_family.family.vcpu_min
    vcpu_max   = data.cloudtemple_public_cloud_vm_instance_family.family.vcpu_max
    ram_min_gb = data.cloudtemple_public_cloud_vm_instance_family.family.ram_min_gb
    ram_max_gb = data.cloudtemple_public_cloud_vm_instance_family.family.ram_max_gb
  }
}

# The priced billing SKUs (vCPU and RAM) of the family: name => unit price.
output "family_pricing" {
  value = {
    for sku in data.cloudtemple_public_cloud_vm_instance_family.family.skus :
    sku.name => sku.price
  }
}
