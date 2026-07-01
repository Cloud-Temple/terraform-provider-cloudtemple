# List the disks of a VM.
data "cloudtemple_public_cloud_vm_disks" "all" {
  virtual_machine_id = "00000000-0000-0000-0000-000000000000"
}

output "data_disk_sizes" {
  value = [for d in data.cloudtemple_public_cloud_vm_disks.all.disks : d.size_gb if !d.is_primary]
}
