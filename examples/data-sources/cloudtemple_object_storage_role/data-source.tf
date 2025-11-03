# Get a specific role by name
data "cloudtemple_object_storage_role" "read" {
  name = "READ"
}

# Output the role
output "read_role" {
  value = data.cloudtemple_object_storage_role.read.name
}

# Use in ACL entry
resource "cloudtemple_object_storage_acl_entry" "example" {
  bucket          = "my-bucket"
  role            = data.cloudtemple_object_storage_role.read.name
  storage_account = "my-storage-account"
}
