data "cloudtemple_compute_machine_manager" "vstack01" {
  name = "vc-vstack-001-t0001"
}

data "cloudtemple_compute_virtual_datacenter" "th3s" {
  name               = "DC-TH3S"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack01.id
}

data "cloudtemple_compute_host_cluster" "clu001" {
  name          = "clu001-ucs12"
  datacenter_id = data.cloudtemple_compute_virtual_datacenter.th3s.id
}

data "cloudtemple_compute_datastore_cluster" "sdrs001" {
  name          = "sdrs001-LIVE_"
  datacenter_id = data.cloudtemple_compute_virtual_datacenter.th3s.id
}

data "cloudtemple_compute_content_library" "cl001" {
  name = "local-vc-vstack-001-t0001"
}

data "cloudtemple_compute_content_library_item" "ubuntu-cloudinit" {
  content_library_id = data.cloudtemple_compute_content_library.cl001.id
  name               = "ubuntu-22.04.1-desktop-amd64"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform-example-controller"
  power_state = "on"

  memory               = 8 * 1024 * 1024 * 1024
  cpu                  = 4
  num_cores_per_socket = 1

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.th3s.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.clu001.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.sdrs001.id
  guest_operating_system_moref = "ubuntu64Guest"
}

resource "cloudtemple_compute_virtual_controller" "bar" {
  virtual_machine_id      = cloudtemple_compute_virtual_machine.foo.id
  type                    = "CD/DVD"
  content_library_item_id = data.cloudtemple_compute_content_library_item.ubuntu-cloudinit.id
  connected               = true
  mounted                 = true
}

resource "cloudtemple_compute_virtual_controller" "baz" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.pbt-crashtest.id
  type               = "SCSI"
  sub_type           = "ParaVirtual"
}