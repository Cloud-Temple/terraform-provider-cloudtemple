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