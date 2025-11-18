data "cloudtemple_object_storage_bucket" "example" {
  name = "my-bucket-name"
}

output "bucket_id" {
  value = data.cloudtemple_object_storage_bucket.example
}
