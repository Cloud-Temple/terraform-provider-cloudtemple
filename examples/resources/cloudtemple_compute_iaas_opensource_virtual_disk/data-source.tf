data "cloudtemple_compute_iaas_opensource_availability_zone" "AZ06" {
  name = "AZ06"
}

data "cloudtemple_compute_iaas_opensource_storage_repository" "sr001-clu001-t0001-az05-r-flh1-data13" {
  name               = "sr002-clu001-t0001-az05-r-flh1-data13"
  type               = "lvmohba"
  shared             = true
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.AZ06.id
}

data "cloudtemple_compute_iaas_opensource_virtual_machine" "OPENIAAS-TERRAFORM-01" {
  name               = "OPENIAAS-TERRAFORM-01"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.AZ06.id
}