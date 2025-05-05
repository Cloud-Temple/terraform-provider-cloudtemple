# This example demonstrates how to use the data source to get backup policies from an Open IaaS infrastructure.
# The data source is used to fetch all backup policies or filter them by various criteria.

# Basic example - Retrieve all backup policies
data "cloudtemple_backup_iaas_opensource_policies" "all" {
}

output "all_policies" {
  value = data.cloudtemple_backup_iaas_opensource_policies.all.policies
}

# Example with name filter
data "cloudtemple_backup_iaas_opensource_policies" "by_name" {
  name = "daily-backup"
}

output "policies_by_name" {
  value = data.cloudtemple_backup_iaas_opensource_policies.by_name.policies
}

# Example with machine_manager_id filter
data "cloudtemple_backup_iaas_opensource_policies" "by_machine_manager" {
  machine_manager_id = "machine_manager_id"
}

output "policies_by_machine_manager" {
  value = data.cloudtemple_backup_iaas_opensource_policies.by_machine_manager.policies
}

# Example with virtual_machine_id filter
data "cloudtemple_backup_iaas_opensource_policies" "by_virtual_machine" {
  virtual_machine_id = "virtual_machine_id"
}

output "policies_by_virtual_machine" {
  value = data.cloudtemple_backup_iaas_opensource_policies.by_virtual_machine.policies
}

# Example with multiple filters
data "cloudtemple_backup_iaas_opensource_policies" "filtered" {
  name               = "daily-backup"
  machine_manager_id = "machine_manager_id"
}

output "filtered_policies" {
  value = data.cloudtemple_backup_iaas_opensource_policies.filtered.policies
}
