# This example demonstrates how to use the data source to get a specific backup from an Open IaaS infrastructure.
# The data source is used to fetch a backup by its ID.

# Example with the ID of the backup
data "cloudtemple_backup_iaas_opensource_backup" "example" {
  id = "backup_id"
}

output "backup_details" {
  value = data.cloudtemple_backup_iaas_opensource_backup.example
}
