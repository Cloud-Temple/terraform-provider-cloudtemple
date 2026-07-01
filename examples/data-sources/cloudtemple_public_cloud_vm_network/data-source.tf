# Retrieve a Public Cloud VM Instances network by name.
data "cloudtemple_public_cloud_vm_network" "lan" {
  name = "LAN01"
}

# Or by id.
data "cloudtemple_public_cloud_vm_network" "by_id" {
  id = "003950b2-d03d-47f9-a66d-7e10397803de"
}

# The network id is what a network adapter attaches to.
output "network_id" {
  value = data.cloudtemple_public_cloud_vm_network.lan.id
}
