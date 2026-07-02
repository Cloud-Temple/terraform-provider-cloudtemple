# Retrieve a single VM instance by id.
data "cloudtemple_public_cloud_vm_instance" "web" {
  id = "00000000-0000-0000-0000-000000000000"
}

output "web" {
  value = {
    name   = data.cloudtemple_public_cloud_vm_instance.web.name
    status = data.cloudtemple_public_cloud_vm_instance.web.status
    vcpu   = data.cloudtemple_public_cloud_vm_instance.web.vcpu
    ram_gb = data.cloudtemple_public_cloud_vm_instance.web.ram_gb
  }
}
