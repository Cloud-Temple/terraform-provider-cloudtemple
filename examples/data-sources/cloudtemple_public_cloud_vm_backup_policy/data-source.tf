# Retrieve a backup policy by name. A policy id is required to create a VM
# (backup_policy_id).
data "cloudtemple_public_cloud_vm_backup_policy" "daily" {
  name = "Daily backup — 30 days"
}

output "policy" {
  value = {
    id        = data.cloudtemple_public_cloud_vm_backup_policy.daily.id
    retention = data.cloudtemple_public_cloud_vm_backup_policy.daily.retention
  }
}
