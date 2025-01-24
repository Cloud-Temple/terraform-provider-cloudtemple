# This example demonstrates how to use the data source to get the policy details.
# The data source is used to fetch the policy details based on the policy ID or name.

# Exemple with the ID of the policy.
data "cloudtemple_backup_iaas_opensource_policy" "policy-1" {
  id = "policy_id"
}

output "policy-1" {
  value = data.cloudtemple_backup_iaas_opensource_policy.policy-1
}

# Exemple with the name of the policy.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_backup_iaas_opensource_policy" "policy-2" {
  name               = "policy_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "policy-2" {
  value = data.cloudtemple_backup_iaas_opensource_policy.policy-2
}