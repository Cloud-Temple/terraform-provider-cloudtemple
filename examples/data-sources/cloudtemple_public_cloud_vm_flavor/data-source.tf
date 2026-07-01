# Retrieve a Public Cloud VM Instances flavor by name.
data "cloudtemple_public_cloud_vm_flavor" "micro" {
  name = "dev-micro"
}

output "micro_vcpu" {
  value = data.cloudtemple_public_cloud_vm_flavor.micro.vcpu
}

output "micro_ram_gb" {
  value = data.cloudtemple_public_cloud_vm_flavor.micro.ram_gb
}
