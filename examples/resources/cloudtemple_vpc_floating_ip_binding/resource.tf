# Bind a PRE-EXISTING floating IP to a static IP. The floating IP must already
# be provisioned out-of-band: this resource only manages the association
# (create = bind, destroy = unbind) — it never creates or destroys the floating
# IP, nor manages its description.
resource "cloudtemple_vpc_floating_ip_binding" "example" {
  floating_ip_id = "a1b2c3d4-e5f6-7890-1234-567890abcdef"
  static_ip_id   = cloudtemple_vpc_static_ip.vm.id
}

# Output the bound addresses (read-only, refreshed from the API).
output "floating_ip_address" {
  value = cloudtemple_vpc_floating_ip_binding.example.floating_ip_address
}

output "bound_static_ip_address" {
  value = cloudtemple_vpc_floating_ip_binding.example.static_ip_address
}
