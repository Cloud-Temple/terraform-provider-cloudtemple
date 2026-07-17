variable "virtual_machine_id" {
  type        = string
  description = "The ID of the VM to snapshot."
}

# A point-in-time snapshot taken before a risky change. `name` is immutable —
# renaming recreates the snapshot; reverting a VM to a snapshot is not managed
# by Terraform.
resource "cloudtemple_public_cloud_vm_snapshot" "pre_upgrade" {
  virtual_machine_id = var.virtual_machine_id
  name               = "pre-upgrade"
}

output "snapshot_status" {
  value = cloudtemple_public_cloud_vm_snapshot.pre_upgrade.status
}

output "snapshot_created_at" {
  value = cloudtemple_public_cloud_vm_snapshot.pre_upgrade.created_at
}
