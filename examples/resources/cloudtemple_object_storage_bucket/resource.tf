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
