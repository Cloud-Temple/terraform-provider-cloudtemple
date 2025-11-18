resource "cloudtemple_object_storage_global_access_key" "example" {
  # This resource manages the global access key for object storage
  # Creating this resource will renew the global access key credentials
}

output "access_key_id" {
  value = cloudtemple_object_storage_global_access_key.example.access_key_id
}

output "access_secret_key" {
  value     = cloudtemple_object_storage_global_access_key.example.access_secret_key
  sensitive = true
}
