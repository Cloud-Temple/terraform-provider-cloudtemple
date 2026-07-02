data "cloudtemple_public_cloud_vm_instance_family" "family" {
  name = "Development"
}

# Retrieve the flavors of an instance family.
data "cloudtemple_public_cloud_vm_flavors" "dev" {
  instance_family_id = data.cloudtemple_public_cloud_vm_instance_family.family.id
}

output "dev_sizings" {
  value = [
    for f in data.cloudtemple_public_cloud_vm_flavors.dev.flavors :
    "${f.name}: ${f.vcpu} vCPU / ${f.ram_gb} GB"
  ]
}
