# This example demonstrates how to use the data source to list all replication policies.

# Example 1: List all replication policies
data "cloudtemple_compute_iaas_opensource_replication_policies" "all_policies" {
}

output "all_replication_policies" {
  value = data.cloudtemple_compute_iaas_opensource_replication_policies.all_policies.policies
}
