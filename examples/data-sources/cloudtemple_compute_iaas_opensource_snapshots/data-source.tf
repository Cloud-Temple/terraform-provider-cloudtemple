// Data source to retrieve information about a list of snapshots in CloudTemple IaaS OpenSource.

# This example demonstrates how to use the data source to get all the snapshots of an Open IaaS infrastructure.
data "cloudtemple_compute_iaas_opensource_snapshots" "all-snapshots" {}

output "snapshot" {
  value = data.cloudtemple_compute_iaas_opensource_snapshots.all-snapshots
}

# This example demonstrates how to use the data source to get all the snapshots of a virtual machine.
data "cloudtemple_compute_iaas_opensource_snapshots" "all-snapshots-of-vm" {
  virtual_machine_id = "virtual_machine_id"
}

output "snapshot" {
  value = data.cloudtemple_compute_iaas_opensource_snapshots.all-snapshots-of-vm
}