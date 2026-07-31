# Retrieve all OS images of the tenant.
data "cloudtemple_public_cloud_vm_images" "all" {}

output "image_names" {
  value = [for i in data.cloudtemple_public_cloud_vm_images.all.images : i.name]
}

# Filter in HCL, e.g. only the Linux images.
output "linux_images" {
  value = [
    for i in data.cloudtemple_public_cloud_vm_images.all.images :
    i.name if i.os_family == "linux"
  ]
}
