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

data "cloudtemple_object_storage_role" "read_only" {
  name = "read_only"
}

data "cloudtemple_object_storage_role" "maintainer" {
  name = "maintainer"
}

resource "cloudtemple_object_storage_bucket" "with_acl" {
  name        = "my-bucket-with-acl"
  access_type = "private"

  acl_entry {
    storage_account = data.cloudtemple_object_storage_storage_account.storage_account_1.name
    role            = data.cloudtemple_object_storage_role.read_only.name
  }

  acl_entry {
    storage_account = data.cloudtemple_object_storage_storage_account.storage_account_2.name
    role            = data.cloudtemple_object_storage_role.maintainer.name
  }
}

# Upload a file to the bucket
data "cloudtemple_object_storage_role" "maintainer" {
  name = "maintainer"
}

resource "cloudtemple_object_storage_storage_account" "storage-account-demo-upload" {
  name = "storage-account-demo-upload"
}

resource "cloudtemple_object_storage_bucket" "bucket-demo-upload" {
  name        = "bucket-demo-upload"
  access_type = "public"

  acl_entry {
    storage_account = cloudtemple_object_storage_storage_account.storage-account-demo-upload.name
    role            = data.cloudtemple_object_storage_role.maintainer.name
  }
}

provider "aws" {
  alias  = "cloudtemple_s3"
  region = "eu-west-3"

  access_key = cloudtemple_object_storage_storage_account.storage-account-demo-upload.access_key_id
  secret_key = cloudtemple_object_storage_storage_account.storage-account-demo-upload.access_secret_key

  endpoints {
    s3 = "http://<namespace>.s3.fr1.cloud-temple.com"
  }

  # avoid validating credentials against AWS
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
}

resource "aws_s3_object" "example_file" {
  provider = aws.cloudtemple_s3
  bucket   = cloudtemple_object_storage_bucket.bucket-demo-upload.name
  key      = "hello-world.txt"
  source   = "local-file.txt"
}

data "cloudtemple_object_storage_bucket_files" "bucket-demo-upload_files" {
  depends_on  = [aws_s3_object.example_file]
  bucket_name = cloudtemple_object_storage_bucket.bucket-demo-upload.name
}

# Show the file named "hello-world.txt" uploaded to the bucket
output "pbt_bucket_02_files" {
  value = [for file in data.cloudtemple_object_storage_bucket_files.bucket-demo-upload_files.files : file if file.key == "hello-world.txt"]
}
