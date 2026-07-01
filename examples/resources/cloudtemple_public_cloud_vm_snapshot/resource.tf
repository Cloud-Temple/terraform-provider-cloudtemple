variable "virtual_machine_id" {
  type        = string
  description = "The ID of the VM to snapshot."
}

# A point-in-time snapshot of a VM. Renaming recreates the snapshot; reverting a
# VM to a snapshot is not managed by Terraform.
resource "cloudtemple_public_cloud_vm_snapshot" "backup" {
  virtual_machine_id = var.virtual_machine_id
  name               = "before-upgrade"
}
