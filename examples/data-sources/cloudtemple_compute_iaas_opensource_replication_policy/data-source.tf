# This example demonstrates how to use the data source to get replication policy details.
# The data source is used to fetch the replication policy details based on the policy ID.

data "cloudtemple_compute_iaas_opensource_replication_policy" "policy" {
  id = "replication_policy_id"
}

output "replication_policy" {
  value = data.cloudtemple_compute_iaas_opensource_replication_policy.policy
}
