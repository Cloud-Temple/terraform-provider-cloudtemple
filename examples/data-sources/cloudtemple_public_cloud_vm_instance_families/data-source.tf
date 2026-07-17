# Retrieve all instance families of the tenant.
data "cloudtemple_public_cloud_vm_instance_families" "all" {}

output "family_names" {
  value = [for f in data.cloudtemple_public_cloud_vm_instance_families.all.instance_families : f.name]
}
