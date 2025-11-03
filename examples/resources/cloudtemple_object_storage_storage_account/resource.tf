# Create a storage account
resource "cloudtemple_object_storage_storage_account" "example" {
  name = "my-storage-account"
}

# Output the access credentials
output "access_key_id" {
  value = cloudtemple_object_storage_storage_account.example.access_key_id
}

output "access_secret_key" {
  value     = cloudtemple_object_storage_storage_account.example.access_secret_key
  sensitive = true
}

output "arn" {
  value = cloudtemple_object_storage_storage_account.example.arn
}
