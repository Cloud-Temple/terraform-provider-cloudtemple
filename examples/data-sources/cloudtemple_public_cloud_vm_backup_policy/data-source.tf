# Retrieve a Public Cloud VM Instances backup policy by name.
# A backup policy is required to create a VM (backup_policy_id).
data "cloudtemple_public_cloud_vm_backup_policy" "daily" {
  name = "Daily backup — 30 days"
}

output "daily_retention" {
  value = data.cloudtemple_public_cloud_vm_backup_policy.daily.retention
}
