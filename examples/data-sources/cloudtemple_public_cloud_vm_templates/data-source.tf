# Retrieve all templates (OS images) of the tenant.
data "cloudtemple_public_cloud_vm_templates" "all" {}

output "template_names" {
  value = [for t in data.cloudtemple_public_cloud_vm_templates.all.templates : t.name]
}

# Filter in HCL, e.g. only the Linux images.
output "linux_templates" {
  value = [
    for t in data.cloudtemple_public_cloud_vm_templates.all.templates :
    t.name if t.os_family == "linux"
  ]
}
