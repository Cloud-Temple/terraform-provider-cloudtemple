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

# Create a storage account with ACL entries
data "cloudtemple_object_storage_bucket" "example-bucket-1" {
  name = "my-bucket-1"
}

data "cloudtemple_object_storage_bucket" "example-bucket-2" {
  name = "my-bucket-2"
}

data "cloudtemple_object_storage_role" "read" {
  name = "READ"
}

data "cloudtemple_object_storage_role" "write" {
  name = "WRITE"
}

resource "cloudtemple_object_storage_storage_account" "with_acl" {
  name = "my-storage-account-with-acl"

  acl_entry {
    role   = data.cloudtemple_object_storage_role.read.name
    bucket = data.cloudtemple_object_storage_bucket.example-bucket-1.name
  }

  acl_entry {
    role   = data.cloudtemple_object_storage_role.write.name
    bucket = data.cloudtemple_object_storage_bucket.example-bucket-2.name
  }
}
