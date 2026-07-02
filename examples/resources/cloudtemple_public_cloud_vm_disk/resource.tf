variable "virtual_machine_id" {
  type        = string
  description = "The ID of the VM to attach the data disks to."
}

# A 20 GB data disk with the platform-default storage type and name.
resource "cloudtemple_public_cloud_vm_disk" "data" {
  virtual_machine_id = var.virtual_machine_id
  size               = 20
}

# A named disk on an explicit storage type.
data "cloudtemple_public_cloud_vm_storage_type" "fast" {
  name = "Enterprise"
}

resource "cloudtemple_public_cloud_vm_disk" "logs" {
  virtual_machine_id = var.virtual_machine_id
  size               = 50
  name               = "logs-01"
  storage_type       = data.cloudtemple_public_cloud_vm_storage_type.fast.id
}

# Extending a disk (increasing `size`) is grow-only and requires the VM to be
# stopped (power_state = "off" on the VM). Destroying a disk stops a running
# VM automatically.

output "data_disk_position" {
  value = cloudtemple_public_cloud_vm_disk.data.position
}
