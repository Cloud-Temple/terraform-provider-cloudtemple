data "cloudtemple_compute_iaas_opensource_availability_zone" "AZ06" {
  name = "AZ06"
}

data "cloudtemple_compute_iaas_opensource_host" "xcp-na85-ucs15-az05" {
  name               = "xcp-na85-ucs15-az05"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.AZ06.id
}

data "cloudtemple_compute_iaas_opensource_template" "AlmaLinux8" {
  name               = "AlmaLinux 8"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.AZ06.id
}

data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
  name               = "nobackup"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.AZ06.id
}