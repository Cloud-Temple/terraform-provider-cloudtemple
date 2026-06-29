resource "cloudtemple_compute_iaas_opensource_network_adapter" "VIF-0" {
  # Mandatory parameters
  virtual_machine_id = cloudtemple_compute_iaas_opensource_virtual_machine.OPENIAAS-TERRAFORM-01.id
  network_id         = data.cloudtemple_compute_iaas_opensource_network.VLAN-01.id

  # Optional parameters
  mac_address = "dc:f1:0a:a9:55:05"
  attached    = true
}

resource "cloudtemple_compute_iaas_opensource_network_adapter" "VIF-1" {
  # It is preferable to wait for the previous network adapter to be created before creating the next one to avoid duplicated IDs.
  depends_on = [cloudtemple_compute_iaas_opensource_network_adapter.VIF-0]

  virtual_machine_id = cloudtemple_compute_iaas_opensource_virtual_machine.OPENIAAS-TERRAFORM-01.id
  network_id         = data.cloudtemple_compute_iaas_opensource_network.VLAN-01.id

  attached = true
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
data "cloudtemple_compute_iaas_opensource_network" "DMZ-ADMIN" {
  name = "DMZ-ADMIN" # a VPC-backed network
}

resource "cloudtemple_compute_iaas_opensource_network_adapter" "vpc" {
  virtual_machine_id = cloudtemple_compute_iaas_opensource_virtual_machine.OPENIAAS-TERRAFORM-01.id
  network_id         = data.cloudtemple_compute_iaas_opensource_network.DMZ-ADMIN.id
  ip_address         = "192.168.0.50" # the VPC static IP to assign; omit for auto-assign
}

# The VPC association is reflected back after apply (resolved by the adapter MAC).
output "vpc_static_ip" {
  value = cloudtemple_compute_iaas_opensource_network_adapter.vpc.static_ip_address
}

output "vpc_private_network_id" {
  value = cloudtemple_compute_iaas_opensource_network_adapter.vpc.private_network_id
}