# Look up the catalogue ids by name.
data "cloudtemple_public_cloud_vm_availability_zone" "az" {
  name = "fr1-az01"
}

data "cloudtemple_public_cloud_vm_instance_family" "family" {
  name = "Development"
}

data "cloudtemple_public_cloud_vm_template" "os" {
  name = "Rocky Linux 9"
}

data "cloudtemple_public_cloud_vm_backup_policy" "policy" {
  name = "No Backup"
}

variable "network_id" {
  type        = string
  description = "The ID of the network to attach the VM's first interface to."
}

# A small Rocky Linux VM, booted at creation.
resource "cloudtemple_public_cloud_vm_instance" "web" {
  name                 = "web-01"
  availability_zone_id = data.cloudtemple_public_cloud_vm_availability_zone.az.id
  template_id          = data.cloudtemple_public_cloud_vm_template.os.id
  instance_family_id   = data.cloudtemple_public_cloud_vm_instance_family.family.id
  cpu                  = 2
  memory               = 4
  backup_policy_id     = data.cloudtemple_public_cloud_vm_backup_policy.policy.id
  power_state          = "on"

  network_interfaces {
    device_index = 0
    network_id   = var.network_id
  }
}

output "vm_status" {
  value = cloudtemple_public_cloud_vm_instance.web.status
}
