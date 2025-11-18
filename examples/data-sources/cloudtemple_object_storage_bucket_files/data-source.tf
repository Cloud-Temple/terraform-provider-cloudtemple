# List all files in a bucket
data "cloudtemple_object_storage_bucket_files" "all_files" {
  bucket_name = "my-bucket"
}

output "all_files" {
  value = data.cloudtemple_object_storage_bucket_files.all_files.files
}

# List files in a specific folder
data "cloudtemple_object_storage_bucket_files" "folder_files" {
  bucket_name = "my-bucket"
  folder_path = "documents/reports"
}

output "folder_files" {
  value = data.cloudtemple_object_storage_bucket_files.folder_files.files
}

# Filter files by size
output "large_files" {
  value = [
    for file in data.cloudtemple_object_storage_bucket_files.all_files.files :
    file if file.size > 1000000 # Files larger than 1MB
  ]
}

# Get the latest version of each file
output "latest_versions" {
  value = [
    for file in data.cloudtemple_object_storage_bucket_files.all_files.files : {
      key = file.key
      version = [
        for v in file.versions :
        v if v.is_latest
      ][0]
    }
  ]
}
