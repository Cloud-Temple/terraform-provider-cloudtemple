# Grant READ access to a storage account on a bucket
resource "cloudtemple_object_storage_acl_entry" "read_access" {
  bucket          = "my-bucket"
  role            = "READ"
  storage_account = "my-storage-account"
}

# Grant WRITE access to a storage account on a bucket
resource "cloudtemple_object_storage_acl_entry" "write_access" {
  bucket          = "my-bucket"
  role            = "WRITE"
  storage_account = "my-storage-account"
}

# Grant FULL_CONTROL to a storage account on a bucket
resource "cloudtemple_object_storage_acl_entry" "full_control" {
  bucket          = "my-bucket"
  role            = "FULL_CONTROL"
  storage_account = "my-storage-account"
}

# Use with other resources
resource "cloudtemple_object_storage_bucket" "example" {
  name        = "my-bucket"
  access_type = "private"
}

resource "cloudtemple_object_storage_storage_account" "example" {
  name = "my-storage-account"
}

resource "cloudtemple_object_storage_acl_entry" "example" {
  bucket          = cloudtemple_object_storage_bucket.example.name
  role            = "READ"
  storage_account = cloudtemple_object_storage_storage_account.example.name
}
