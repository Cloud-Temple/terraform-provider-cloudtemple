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

# ---------------------------------------------------------------------------
# Connect an adapter to a VPC private network with a controlled static IP
# ---------------------------------------------------------------------------
# When network_id references a VPC-backed network, set ip_address to assign the
# adapter's VPC static IP. Omit it to let the platform auto-assign the first free
# address (the assigned value is read back into the state, resolved by MAC). The
# value is mutable — changing it relocates the static IP — and destroying the
# adapter releases it. Setting ip_address while network_id is NOT VPC-backed is
# rejected at plan time.
data "cloudtemple_compute_network" "vpc" {
  name = "vpc-private-net" # a VPC-backed network
}

resource "cloudtemple_compute_network_adapter" "vpc" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.web.id
  network_id         = data.cloudtemple_compute_network.vpc.id
  type               = "VMXNET3"
  ip_address         = "192.168.0.50" # the VPC static IP to assign; omit for auto-assign
}