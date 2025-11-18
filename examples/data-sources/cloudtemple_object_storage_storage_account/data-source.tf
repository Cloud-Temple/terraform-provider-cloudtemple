data "cloudtemple_object_storage_storage_account" "example" {
  name = "my-storage-account-name"
}

output "storage_account_id" {
  value = data.cloudtemple_object_storage_storage_account.example
}
