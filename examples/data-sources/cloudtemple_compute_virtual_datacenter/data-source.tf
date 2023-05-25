data "cloudtemple_compute_machine_manager" "vstack-001" {
  name = "vc-vstack-001-t0001"
}

data "cloudtemple_compute_virtual_datacenter" "id" {
  id = "ac33c033-693b-4fc5-9196-26df77291dbb"
}

data "cloudtemple_compute_virtual_datacenter" "name" {
  name               = "DC-EQX6"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}