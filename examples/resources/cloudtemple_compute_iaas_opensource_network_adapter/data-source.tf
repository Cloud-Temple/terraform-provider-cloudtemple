data "cloudtemple_compute_iaas_opensource_availability_zone" "AZ06" {
  name = "AZ06"
}

data "cloudtemple_compute_iaas_opensource_virtual_machine" "OPENIAAS-TERRAFORM-01" {
  name               = "OPENIAAS-TERRAFORM-01"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.AZ06.id
}

data "cloudtemple_compute_iaas_opensource_network" "VLAN-01" {
  name               = "VLAN-01"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.AZ06.id
}