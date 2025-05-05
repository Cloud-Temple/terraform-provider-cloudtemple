// Data source to retrieve information about a snapshot in CloudTemple IaaS OpenSource.

# This example demonstrates how to use the data source to get a snapshot by its ID.
data "cloudtemple_compute_iaas_opensource_snapshot" "snapshot-1" {
  id = "snapshot_id"
}

output "snapshot" {
  value = data.cloudtemple_compute_iaas_opensource_snapshot.snapshot-1
}

# This example demonstrates how to use the data source to get a snapshot by its name and virtual machine ID.
data "cloudtemple_compute_iaas_opensource_snapshot" "snapshot-2" {
  name               = "snapshot_name"
  virtual_machine_id = "virtual_machine_id"
}

output "snapshot" {
  value = data.cloudtemple_compute_iaas_opensource_snapshot.snapshot-2
}