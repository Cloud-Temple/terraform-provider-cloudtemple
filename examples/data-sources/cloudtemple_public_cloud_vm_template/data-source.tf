# Retrieve a template (OS image) by name.
data "cloudtemple_public_cloud_vm_template" "os" {
  name = "Debian 13"
}

output "template" {
  value = {
    id         = data.cloudtemple_public_cloud_vm_template.os.id
    os_family  = data.cloudtemple_public_cloud_vm_template.os.os_family
    os_version = data.cloudtemple_public_cloud_vm_template.os.os_version
  }
}
