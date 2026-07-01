variable "virtual_machine_id" {
  type        = string
  description = "The ID of the VM to attach the data disk to."
}

# A 20 GB data disk. Extending it (increasing `size`) requires the VM to be
# stopped; the disk can only grow, never shrink.
resource "cloudtemple_public_cloud_vm_disk" "data" {
  virtual_machine_id = var.virtual_machine_id
  size               = 20
}
