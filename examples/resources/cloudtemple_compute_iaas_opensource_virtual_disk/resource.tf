# Exemple with a bootable Read only disk
resource "cloudtemple_compute_iaas_opensource_virtual_disk" "openiaas-disk-01" {
  name     = "openiaas-disk-01"
  size     = 2 * 1024 * 1024 * 1024
  mode     = "RO"
  bootable = true

  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.sr001-clu001-t0001-az05-r-flh1-data13.id
  virtual_machine_id    = cloudtemple_compute_iaas_opensource_virtual_machine.OPENIAAS-TERRAFORM-01.id
}

# Exemple with a non-bootable Read/Write disk
resource "cloudtemple_compute_iaas_opensource_virtual_disk" "openiaas-disk-02" {
  name     = "openiaas-disk-02"
  size     = 4 * 1024 * 1024 * 1024
  mode     = "RW"
  bootable = false

  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.sr001-clu001-t0001-az05-r-flh1-data13.id
  virtual_machine_id    = cloudtemple_compute_iaas_opensource_virtual_machine.OPENIAAS-TERRAFORM-01.id
}

# Exemple with a disconnected disk (connected = false)
resource "cloudtemple_compute_iaas_opensource_virtual_disk" "openiaas-disk-03" {
  name      = "openiaas-disk-03"
  size      = 4 * 1024 * 1024 * 1024
  mode      = "RW"
  bootable  = false
  connected = false # The disk is attached but not connected to the VM

  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.sr001-clu001-t0001-az05-r-flh1-data13.id
  virtual_machine_id    = cloudtemple_compute_iaas_opensource_virtual_machine.OPENIAAS-TERRAFORM-01.id
}

# Exemple with a non-bootable Read/Write standalone disk, not attached to any VM
resource "cloudtemple_compute_iaas_opensource_virtual_disk" "openiaas-disk-04" {
  name     = "openiaas-disk-04"
  size     = 4 * 1024 * 1024 * 1024
  mode     = "RW"
  bootable = false

  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.sr001-clu001-t0001-az05-r-flh1-data13.id
}
