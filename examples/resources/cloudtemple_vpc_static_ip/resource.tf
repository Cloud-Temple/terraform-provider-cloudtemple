# Allocate a static IP on a private network with an auto-assigned address,
# bound to a VM network-adapter MAC address.
resource "cloudtemple_vpc_static_ip" "auto" {
  private_network_id = "b2c3d4e5-f678-9012-3456-7890abcdef12"
  mac_address        = "00:50:56:ab:cd:ef"
}

# RECOMMENDED: wire mac_address from the VM's network adapter. This makes the
# static IP depend on the adapter, so `terraform destroy` removes the IP
# association BEFORE it destroys the VM/adapter — which avoids leaving an
# orphaned static IP on the private network.
resource "cloudtemple_vpc_static_ip" "vm" {
  private_network_id = "b2c3d4e5-f678-9012-3456-7890abcdef12"
  mac_address        = cloudtemple_compute_iaas_opensource_network_adapter.example.mac_address
}

# Allocate a static IP with an explicit desired address and a description.
# resource_description is optional and defaults to "Managed by Terraform".
resource "cloudtemple_vpc_static_ip" "explicit" {
  private_network_id   = "b2c3d4e5-f678-9012-3456-7890abcdef12"
  mac_address          = "00:50:56:ab:cd:f0"
  ip_address           = "10.0.1.50"
  resource_description = "Web server production"
}

# Output the (possibly auto-assigned) address and the VPC it belongs to.
output "static_ip_address" {
  value = cloudtemple_vpc_static_ip.auto.ip_address
}

output "static_ip_vpc_id" {
  value = cloudtemple_vpc_static_ip.auto.vpc_id
}
