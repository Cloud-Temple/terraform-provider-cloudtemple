# Retrieve all flavors of the tenant.
data "cloudtemple_public_cloud_vm_flavors" "all" {}

# Or only those of a given instance family.
data "cloudtemple_public_cloud_vm_flavors" "dev" {
  instance_family_id = "287e8d20-4f07-4107-8b86-8642f6864d3c"
}

output "flavor_names" {
  value = [for f in data.cloudtemple_public_cloud_vm_flavors.all.flavors : f.name]
}
