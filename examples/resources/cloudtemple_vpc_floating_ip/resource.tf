# Provision an unbound floating (public) IP. The address is allocated by the API.
resource "cloudtemple_vpc_floating_ip" "egress" {}

# Provision a floating IP with a description (mutable; applied via a PATCH).
resource "cloudtemple_vpc_floating_ip" "described" {
  description = "Production egress IP"
}

# Output the allocated public address.
output "floating_ip_address" {
  value = cloudtemple_vpc_floating_ip.egress.ip_address
}
