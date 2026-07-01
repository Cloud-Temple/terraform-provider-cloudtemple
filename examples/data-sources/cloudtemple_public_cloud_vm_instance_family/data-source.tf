# Retrieve a Public Cloud VM Instances instance family by name.
data "cloudtemple_public_cloud_vm_instance_family" "general" {
  name = "General Purpose"
}

output "general_vcpu_max" {
  value = data.cloudtemple_public_cloud_vm_instance_family.general.vcpu_max
}
