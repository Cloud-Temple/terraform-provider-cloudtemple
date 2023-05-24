data "cloudtemple_compute_machine_manager" "vstack-001" {
  name = "vc-vstack-001-t0001"
}

# Read an datastore cluster using its ID
data "cloudtemple_compute_datastore_cluster" "id" {
  id = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
}

# Read an datastore cluster using its name
data "cloudtemple_compute_datastore_cluster" "name" {
  name = "sdrs001-LIVE_KOUKOU"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}