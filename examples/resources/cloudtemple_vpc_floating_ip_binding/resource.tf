# Bind a public floating IP to a private static IP — the "expose a VM publicly"
# action (VM adapter MAC -> static IP -> floating IP). The floating IP and the
# static IP are provisioned separately; this resource only manages the binding and
# leaves the floating IP intact on destroy.
resource "cloudtemple_vpc_floating_ip_binding" "public" {
  floating_ip_id = cloudtemple_vpc_floating_ip.egress.id
  static_ip_id   = cloudtemple_vpc_static_ip.vm.id
}

# The public address the binding exposes.
output "public_ip" {
  value = cloudtemple_vpc_floating_ip_binding.public.floating_ip_address
}
