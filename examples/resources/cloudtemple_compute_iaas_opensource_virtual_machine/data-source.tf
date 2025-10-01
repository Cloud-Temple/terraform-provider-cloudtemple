data "cloudtemple_compute_iaas_opensource_availability_zone" "az05" {
  name = "az05"
}

data "cloudtemple_compute_iaas_opensource_host" "xcp-na85-ucs15-az05" {
  name               = "xcp-na85-ucs15-az05"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.az05.id
}

data "cloudtemple_compute_iaas_opensource_storage_repository" "sr011-clu001-t0001-az05-r-flh1-data13" {
  name               = "sr011-clu001-t0001-az05-r-flh1-data13"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.az05.id
}

data "cloudtemple_compute_iaas_opensource_template" "AlmaLinux8" {
  name               = "AlmaLinux 8"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.az05.id
}

data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
  name               = "nobackup"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.az05.id
}