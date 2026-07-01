variable "virtual_machine_id" {
  type        = string
  description = "The ID of the VM to attach the network adapter to."
}

# Resolve a network from the catalogue by name.
data "cloudtemple_public_cloud_vm_network" "lan" {
  name = "LAN01"
}

# Attach a network adapter as eth1. Re-pointing it to another network
# (changing network_id) and destroying it both require the VM to be stopped.
resource "cloudtemple_public_cloud_vm_network_adapter" "eth1" {
  virtual_machine_id = var.virtual_machine_id
  device_index       = 1
  network_id         = data.cloudtemple_public_cloud_vm_network.lan.id

  # ip_address is only honoured on VPC networks (ignored on Private Backbone).
  # ip_address = "10.0.0.5"
}
