# This example demonstrates how to use the data source to get backups from an Open IaaS infrastructure.
# The data source is used to fetch all backups or filter them by various criteria.

# Basic example - Retrieve all backups
data "cloudtemple_backup_iaas_opensource_backups" "all" {
}

output "all_backups" {
  value = data.cloudtemple_backup_iaas_opensource_backups.all.backups
}

# Example with machine_manager_id filter
data "cloudtemple_backup_iaas_opensource_backups" "by_machine_manager" {
  machine_manager_id = "machine_manager_id"
}

output "backups_by_machine_manager" {
  value = data.cloudtemple_backup_iaas_opensource_backups.by_machine_manager.backups
}

# Example with virtual_machine_id filter
data "cloudtemple_backup_iaas_opensource_backups" "by_virtual_machine" {
  virtual_machine_id = "virtual_machine_id"
}

output "backups_by_virtual_machine" {
  value = data.cloudtemple_backup_iaas_opensource_backups.by_virtual_machine.backups
}

# Example with deleted filter
data "cloudtemple_backup_iaas_opensource_backups" "include_deleted" {
  deleted = true
}

output "backups_including_deleted" {
  value = data.cloudtemple_backup_iaas_opensource_backups.include_deleted.backups
}

# Example with multiple filters
data "cloudtemple_backup_iaas_opensource_backups" "filtered" {
  machine_manager_id = "machine_manager_id"
  deleted            = false
}

output "filtered_backups" {
  value = data.cloudtemple_backup_iaas_opensource_backups.filtered.backups
}
