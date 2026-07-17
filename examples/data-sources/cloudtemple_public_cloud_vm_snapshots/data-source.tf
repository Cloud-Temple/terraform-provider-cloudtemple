variable "virtual_machine_id" {
  type        = string
  description = "The ID of the VM whose snapshots are listed."
}

# List the snapshots of a VM.
data "cloudtemple_public_cloud_vm_snapshots" "vm" {
  virtual_machine_id = var.virtual_machine_id
}

output "snapshot_names" {
  value = [for s in data.cloudtemple_public_cloud_vm_snapshots.vm.snapshots : s.name]
}
