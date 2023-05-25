data "cloudtemple_compute_machine_manager" "vstack-001" {
  name = "vc-vstack-001-t0001"
}

data "cloudtemple_compute_host_cluster" "id" {
  id = "dde72065-60f4-4577-836d-6ea074384d62"
}

data "cloudtemple_compute_host_cluster" "name" {
  name               = "clu002-ucs01_FLO"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}