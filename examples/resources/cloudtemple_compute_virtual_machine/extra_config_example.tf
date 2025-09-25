# Example: Virtual Machine with Extra Config for CoreOS/Ignition
resource "cloudtemple_compute_virtual_machine" "coreos_vm" {
  name = "coreos-ignition-vm"

  # Basic VM configuration
  datacenter_id   = var.datacenter_id
  host_cluster_id = var.host_cluster_id
  datastore_id    = var.datastore_id

  memory = 2147483648 # 2GB
  cpu    = 2

  guest_operating_system_moref = var.coreos_guest_os_moref
  power_state                  = "on"

  # Extra configuration parameters for advanced VM settings
  extra_config = {
    # Ignition configuration for CoreOS
    "guestinfo.ignition.config.data" = base64encode(jsonencode({
      ignition = {
        version = "3.3.0"
      }
      passwd = {
        users = [{
          name = "core"
          ssh_authorized_keys = [
            "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
          ]
        }]
      }
      systemd = {
        units = [{
          name    = "docker.service"
          enabled = true
        }]
      }
    }))

    "guestinfo.ignition.config.data.encoding"  = "base64"
    "guestinfo.afterburn.initrd.network-kargs" = "ip=dhcp"

    # Performance optimization for virtualized environments
    "stealclock.enable" = "TRUE"

    # Disk configuration
    "disk.enableUUID" = "TRUE"

    # PCI Passthrough configuration (if needed)
    "pciPassthru.use64BitMMIO"    = "TRUE"
    "pciPassthru.64bitMMioSizeGB" = "64"
  }

  tags = {
    environment = "production"
    os_type     = "coreos"
    created_by  = "terraform"
  }
}

# Variables
variable "datacenter_id" {
  description = "The ID of the datacenter"
  type        = string
}

variable "host_cluster_id" {
  description = "The ID of the host cluster"
  type        = string
}

variable "datastore_id" {
  description = "The ID of the datastore"
  type        = string
}

variable "coreos_guest_os_moref" {
  description = "The guest OS moref for CoreOS"
  type        = string
  default     = "coreos64Guest"
}
