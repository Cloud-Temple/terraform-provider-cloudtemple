# List all available object storage roles
data "cloudtemple_object_storage_roles" "example" {}

# Output the roles
output "available_roles" {
  value = data.cloudtemple_object_storage_roles.example.roles
}
