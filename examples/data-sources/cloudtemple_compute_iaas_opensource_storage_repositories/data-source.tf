# This example demonstrates how to use the data source to get storage repositories from an Open IaaS infrastructure.
# The data source is used to fetch all storage repositories or filter them by various criteria.

# Basic example - Retrieve all storage repositories
data "cloudtemple_compute_iaas_opensource_storage_repositories" "all" {
}

output "all_storage_repositories" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repositories.all.storage_repositories
}

# Example with machine_manager_id filter
data "cloudtemple_compute_iaas_opensource_storage_repositories" "by_machine_manager" {
  machine_manager_id = "machine_manager_id"
}

output "storage_repositories_by_machine_manager" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repositories.by_machine_manager.storage_repositories
}

# Example with pool_id filter
data "cloudtemple_compute_iaas_opensource_storage_repositories" "by_pool" {
  pool_id = "pool_id"
}

output "storage_repositories_by_pool" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repositories.by_pool.storage_repositories
}

# Example with host_id filter
data "cloudtemple_compute_iaas_opensource_storage_repositories" "by_host" {
  host_id = "host_id"
}

output "storage_repositories_by_host" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repositories.by_host.storage_repositories
}

# Example with type filter
data "cloudtemple_compute_iaas_opensource_storage_repositories" "by_type" {
  type = "nfs" # Available values: ext, lvm, lvmoiscsi, lvmohba, nfs, smb, iso, nfs_iso, cifs
}

output "storage_repositories_by_type" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repositories.by_type.storage_repositories
}

# Example with shared filter
data "cloudtemple_compute_iaas_opensource_storage_repositories" "shared_only" {
  shared = true
}

output "shared_storage_repositories" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repositories.shared_only.storage_repositories
}

# Example with multiple filters
data "cloudtemple_compute_iaas_opensource_storage_repositories" "filtered" {
  machine_manager_id = "machine_manager_id"
  type               = "nfs"
  shared             = true
}

output "filtered_storage_repositories" {
  value = data.cloudtemple_compute_iaas_opensource_storage_repositories.filtered.storage_repositories
}