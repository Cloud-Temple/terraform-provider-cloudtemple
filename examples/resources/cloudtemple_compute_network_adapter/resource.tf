data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "web" {
  name = "hello-world"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

data "cloudtemple_compute_network" "vlan" {
  name = "VLAN_201"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.web.id
  network_id         = data.cloudtemple_compute_network.vlan.id
  type               = "VMXNET3"
  mac_type           = "ASSIGNED"
}