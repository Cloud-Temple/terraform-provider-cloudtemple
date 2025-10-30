data "cloudtemple_object_storage_storage_accounts" "example" {}

output "all_storage_accounts" {
  value = data.cloudtemple_object_storage_storage_accounts.example.storage_accounts
}

output "storage_account_count" {
  value = length(data.cloudtemple_object_storage_storage_accounts.example.storage_accounts)
}
