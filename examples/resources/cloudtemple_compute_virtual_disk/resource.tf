data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform-example-disk"
  power_state = "off"

  memory                 = 8 * 1024 * 1024 * 1024
  cpu                    = 2
  num_cores_per_socket   = 1
  cpu_hot_add_enabled    = true
  cpu_hot_remove_enabled = true
  memory_hot_add_enabled = true

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

resource "cloudtemple_compute_virtual_disk" "foo" {
  virtual_machine_id   = cloudtemple_compute_virtual_machine.foo.id
  provisioning_type    = "dynamic"
  disk_mode            = "persistent"
  capacity             = 8 * 1024 * 1024 * 1024
  datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.koukou.id
}