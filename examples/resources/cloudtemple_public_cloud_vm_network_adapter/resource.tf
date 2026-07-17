variable "virtual_machine_id" {
  type        = string
  description = "The ID of the VM to attach the network adapters to."
}

# --- Private Backbone network -------------------------------------------

data "cloudtemple_public_cloud_vm_network" "lan" {
  name = "LAN01"
}

# Re-pointing the adapter to another network (changing network_id) requires
# the VM to be stopped; destroying it stops a running VM automatically.
resource "cloudtemple_public_cloud_vm_network_adapter" "eth1" {
  virtual_machine_id = var.virtual_machine_id
  device_index       = 1
  network_id         = data.cloudtemple_public_cloud_vm_network.lan.id
}

# --- VPC network ---------------------------------------------------------

# A network with a non-empty `vpc` block is a VPC network.
data "cloudtemple_public_cloud_vm_network" "vpc_net" {
  name = "my-vpc-private-network"
}

resource "cloudtemple_public_cloud_vm_network_adapter" "eth2" {
  virtual_machine_id = var.virtual_machine_id
  device_index       = 2
  network_id         = data.cloudtemple_public_cloud_vm_network.vpc_net.id

  # Optional static IPv4, registered on the VPC private network. Only honoured
  # on VPC networks (ignored on Private Backbone, with a warning); when omitted
  # the platform assigns one. Write-only: the observed address is `ipv4_address`.
  ip_address = "10.0.0.50"
}

output "eth2_type" {
  value = cloudtemple_public_cloud_vm_network_adapter.eth2.type
}
