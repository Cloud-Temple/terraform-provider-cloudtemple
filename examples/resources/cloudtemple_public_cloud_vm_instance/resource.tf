# Resolve the catalogue by name.
data "cloudtemple_public_cloud_vm_availability_zone" "az" {
  name = "fr1-az01"
}

data "cloudtemple_public_cloud_vm_instance_family" "family" {
  name = "Development"
}

data "cloudtemple_public_cloud_vm_template" "os" {
  name = "Debian 13"
}

data "cloudtemple_public_cloud_vm_backup_policy" "policy" {
  name = "No Backup"
}

# The inline os_network_adapter block only accepts Private Backbone networks;
# attach VPC networks with the cloudtemple_public_cloud_vm_network_adapter
# resource.
data "cloudtemple_public_cloud_vm_network" "lan" {
  name = "LAN01"
}

# A small VM, booted at creation, bootstrapped with cloud-init.
resource "cloudtemple_public_cloud_vm_instance" "web" {
  name                 = "web-01"
  availability_zone_id = data.cloudtemple_public_cloud_vm_availability_zone.az.id
  template_id          = data.cloudtemple_public_cloud_vm_template.os.id
  instance_family_id   = data.cloudtemple_public_cloud_vm_instance_family.family.id
  cpu                  = 2
  memory               = 4
  backup_policy_id     = data.cloudtemple_public_cloud_vm_backup_policy.policy.id
  power_state          = "on"

  os_network_adapter {
    device_index = 0
    network_id   = data.cloudtemple_public_cloud_vm_network.lan.id
  }

  cloud_init = {
    cloud_config = <<-EOT
      #cloud-config
      hostname: web-01
      packages:
        - nginx
    EOT
  }
}

# Day-2 operations, each in its own apply:
# - stop/start: toggle power_state between "on" and "off";
# - resize: change cpu/memory together with power_state = "off";
# - grow the system disk (grow-only, VM stopped) by declaring:
#   os_disk {
#     size_gb = 45
#   }

output "web_status" {
  value = cloudtemple_public_cloud_vm_instance.web.status
}

output "web_os_disk_size_gb" {
  value = one(cloudtemple_public_cloud_vm_instance.web.os_disk[*].size_gb)
}
