# Retrieve all backup policies of the tenant.
data "cloudtemple_public_cloud_vm_backup_policies" "all" {}

output "policies" {
  value = [
    for p in data.cloudtemple_public_cloud_vm_backup_policies.all.backup_policies :
    "${p.name} (${coalesce(p.retention, 0)} restore points)"
  ]
}
