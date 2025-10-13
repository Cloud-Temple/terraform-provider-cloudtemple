# This example demonstrates how to create a replication policy for Open IaaS.
# A replication policy defines the interval at which VM replication occurs.

# First, get the storage repository where replicated VMs will be stored
data "cloudtemple_compute_iaas_opensource_storage_repository" "replication_target" {
  name               = "target_storage_repository_name"
  machine_manager_id = "availability_zone_id"
}

# Example 1: Create a replication policy that runs every 6 hours
resource "cloudtemple_compute_iaas_opensource_replication_policy" "policy_hourly" {
  name                  = "replication-policy-6h"
  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.replication_target.id

  interval {
    hours = 6
  }
}

# Example 2: Create a replication policy that runs every 30 minutes
resource "cloudtemple_compute_iaas_opensource_replication_policy" "policy_minutes" {
  name                  = "replication-policy-30m"
  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.replication_target.id

  interval {
    minutes = 30
  }
}

# You can then associate this policy with a virtual machine using the replication_policy_id attribute
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vm_with_replication" {
  name        = "vm-with-replication"
  template_id = "template_id"
  cpu         = 2
  memory      = 4 * 1024 * 1024 * 1024
  power_state = "on"

  # Associate the replication policy
  replication_policy_id = cloudtemple_compute_iaas_opensource_replication_policy.policy_hourly.id

  # ... other VM configuration ...
}

# Output the replication policy details
output "replication_policy" {
  value = {
    id                 = cloudtemple_compute_iaas_opensource_replication_policy.policy_hourly.id
    name               = cloudtemple_compute_iaas_opensource_replication_policy.policy_hourly.name
    pool_id            = cloudtemple_compute_iaas_opensource_replication_policy.policy_hourly.pool_id
    machine_manager_id = cloudtemple_compute_iaas_opensource_replication_policy.policy_hourly.machine_manager_id
  }
}
