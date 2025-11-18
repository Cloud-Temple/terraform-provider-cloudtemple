data "cloudtemple_object_storage_buckets" "example" {}

output "all_buckets" {
  value = data.cloudtemple_object_storage_buckets.example.buckets
}

output "bucket_count" {
  value = length(data.cloudtemple_object_storage_buckets.example.buckets)
}
