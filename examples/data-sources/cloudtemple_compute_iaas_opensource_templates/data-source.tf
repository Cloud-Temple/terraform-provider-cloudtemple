# This example demonstrates how to use the data source to get templates from an Open IaaS infrastructure.
# The data source is used to fetch all templates or filter them by machine manager ID or pool ID.

# Basic example - Retrieve all templates
data "cloudtemple_compute_iaas_opensource_templates" "all" {
}

output "all_templates" {
  value = data.cloudtemple_compute_iaas_opensource_templates.all.templates
}

# Example with machine_manager_id filter
data "cloudtemple_compute_iaas_opensource_templates" "by_machine_manager" {
  machine_manager_id = "machine_manager_id"
}

output "templates_by_machine_manager" {
  value = data.cloudtemple_compute_iaas_opensource_templates.by_machine_manager.templates
}

# Example with pool_id filter
data "cloudtemple_compute_iaas_opensource_templates" "by_pool" {
  pool_id = "pool_id"
}

output "templates_by_pool" {
  value = data.cloudtemple_compute_iaas_opensource_templates.by_pool.templates
}

# Example with both filters
data "cloudtemple_compute_iaas_opensource_templates" "filtered" {
  machine_manager_id = "machine_manager_id"
  pool_id            = "pool_id"
}

output "filtered_templates" {
  value = data.cloudtemple_compute_iaas_opensource_templates.filtered.templates
}