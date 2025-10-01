resource "cloudtemple_compute_iaas_opensource_virtual_machine" "pbt-openiaas-01" {
  name = "OPENIAAS-TERRAFORM-01"

  power_state = "on"
  host_id     = data.cloudtemple_compute_iaas_opensource_host.xcp-na85-ucs15-az05.id
  template_id = data.cloudtemple_compute_iaas_opensource_template.AlmaLinux8.id

  memory               = 8 * 1024 * 1024 * 1024
  cpu                  = 4
  num_cores_per_socket = 2

  secure_boot       = false
  auto_power_on     = true
  high_availability = "best-effort"

  # Define an os_network_adapter block for each network adapter in the template 
  os_network_adapter {
    network_id      = data.cloudtemple_compute_iaas_opensource_network.p-vlan-01.id
    mac_address     = "c2:db:4f:15:41:3e"
    tx_checksumming = true
    attached        = true
  }

  # Define an os_disk block for each virtual disk in the template
  os_disk {
    name                  = "data-disk-01"
    connected             = true
    size                  = 14 * 1024 * 1024 * 1024
    storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.sr011-clu001-t0001-az05-r-flh1-data13.id
  }

  # Set a list of backup policies to apply to the virtual machine. Defining at least one backup policy is mandatory to power on the VM.
  backup_sla_policies = [
    data.cloudtemple_backup_iaas_opensource_policy.nobackup.id
  ]

  # Define the boot order of the virtual machine. The boot order is a list of strings that represent the boot devices.
  boot_order = [
    "Hard-Drive",
    "DVD-Drive",
    # "Network"
  ]

  # Mount an ISO file to the virtual machine, such as guest tools to install it if needed.
  mount_iso = data.cloudtemple_compute_iaas_opensource_virtual_disk.guest-tools.id

  # Add key-value tags to the virtual machine.
  tags = {
    environment = "development"
  }

  # Add cloud_init settings to the virtual machine.
  cloud_init = {
    cloud_config   = filebase64("./cloud-init/cloud-config.yml")
    network_config = filebase64("./cloud-init/network-config.yml")
  }
}