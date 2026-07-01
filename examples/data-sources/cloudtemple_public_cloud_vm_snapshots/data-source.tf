# List the snapshots of a VM.
data "cloudtemple_public_cloud_vm_snapshots" "all" {
  virtual_machine_id = "00000000-0000-0000-0000-000000000000"
}

output "snapshot_names" {
  value = [for s in data.cloudtemple_public_cloud_vm_snapshots.all.snapshots : s.name]
}
