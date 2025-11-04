# Create a private bucket
resource "cloudtemple_object_storage_bucket" "private" {
  name        = "my-private-bucket"
  access_type = "private"
}

# Create a public bucket
resource "cloudtemple_object_storage_bucket" "public" {
  name        = "my-public-bucket"
  access_type = "public"
}

# Create a custom access bucket with whitelist
resource "cloudtemple_object_storage_bucket" "custom" {
  name        = "my-custom-bucket"
  access_type = "custom"
  whitelist = [
    "10.0.0.0/8",
    "192.168.1.0/24"
  ]
}

# Create a bucket with versioning enabled
resource "cloudtemple_object_storage_bucket" "versioned" {
  name        = "my-versioned-bucket"
  access_type = "public"
  versioning  = "Enabled"
}

# Output bucket information
output "bucket_endpoint" {
  value = cloudtemple_object_storage_bucket.private.endpoint
}

output "bucket_id" {
  value = cloudtemple_object_storage_bucket.private.id
}

# Create a bucket with acl entries
data "cloudtemple_object_storage_storage_account" "storage_account_1" {
  name = "storage-account-1"
}

data "cloudtemple_object_storage_storage_account" "storage_account_1" {
  name = "storage-account-2"
}

data "cloudtemple_object_storage_role" "read" {
  name = "READ"
}

data "cloudtemple_object_storage_role" "write" {
  name = "WRITE"
}

resource "cloudtemple_object_storage_bucket" "with_acl" {
  name        = "my-bucket-with-acl"
  access_type = "private"

  acl_entry {
    storage_account = data.cloudtemple_object_storage_storage_account.storage_account_1.name
    role            = data.cloudtemple_object_storage_role.read.name
  }

  acl_entry {
    storage_account = data.cloudtemple_object_storage_storage_account.storage_account_2.name
    role            = data.cloudtemple_object_storage_role.write.name
  }
}
