# Retrieve all templates of the tenant.
data "cloudtemple_public_cloud_vm_templates" "all" {}

output "template_names" {
  value = [for t in data.cloudtemple_public_cloud_vm_templates.all.templates : t.name]
}
