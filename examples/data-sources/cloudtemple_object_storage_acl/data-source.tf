# List storage accounts with access to a specific bucket
data "cloudtemple_object_storage_acl" "bucket_acls" {
  bucket_name = "my-bucket"
}

output "bucket_storage_accounts" {
  value = data.cloudtemple_object_storage_acl.bucket_acls.acls
}

# List buckets accessible by a specific storage account
data "cloudtemple_object_storage_acl" "account_acls" {
  storage_account_name = "my-storage-account"
}

output "account_buckets" {
  value = data.cloudtemple_object_storage_acl.account_acls.acls
}

# Filter ACLs by role
output "owners_only" {
  value = [
    for acl in data.cloudtemple_object_storage_acl.bucket_acls.acls :
    acl if acl.role == "owner"
  ]
}
