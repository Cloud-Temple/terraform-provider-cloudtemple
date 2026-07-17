variable "virtual_machine_id" {
  type        = string
  description = "The ID of the VM whose disks are listed."
}

# List the disks (system and data) of a VM.
data "cloudtemple_public_cloud_vm_disks" "vm" {
  virtual_machine_id = var.virtual_machine_id
}

output "system_disk_size_gb" {
  value = one([for d in data.cloudtemple_public_cloud_vm_disks.vm.disks : d.size_gb if d.is_primary])
}

output "data_disks" {
  value = [for d in data.cloudtemple_public_cloud_vm_disks.vm.disks : d.name if !d.is_primary]
}
