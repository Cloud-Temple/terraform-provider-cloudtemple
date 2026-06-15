# ct-test scenario `vm` — OpenIaaS VM full lifecycle via Terraform.
#
# Mirrors (and goes beyond) the API `compute_lifecycle` cycle, but exercised through
# the real provider: deploy an Ubuntu VM FROM THE MARKETPLACE, attach a data disk,
# and drive power (start/stop) — with a convergence check (empty plan) between each
# step and a clean destroy at the end. Driven by scripts/ct-test.sh:
#
#   scripts/ct-test.sh tf vm
#
# The substrate (availability zone, storage repository, network, backup policy) is
# DISCOVERED dynamically (no hard-coded tenant ids), so the same config runs on any
# OpenIaaS tenant — like the API cycle's read-then-pick approach. Only the
# marketplace image name is a variable (catalogs differ); see variables.tf.

terraform {
  required_providers {
    cloudtemple = {
      source = "Cloud-Temple/cloudtemple"
    }
  }
}

# Credentials and host come from the environment (CLOUDTEMPLE_CLIENT_ID /
# CLOUDTEMPLE_SECRET_ID / CLOUDTEMPLE_HTTP_ADDR / CLOUDTEMPLE_HTTP_SCHEME), exported
# by scripts/ct-test.sh load_creds.
provider "cloudtemple" {}

# --- substrate discovery (dynamic; first usable of each, like the API cycle) ------

data "cloudtemple_compute_iaas_opensource_availability_zones" "all" {}

data "cloudtemple_compute_iaas_opensource_storage_repositories" "all" {
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zones.all.availability_zones[0].id
}

data "cloudtemple_compute_iaas_opensource_networks" "all" {
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zones.all.availability_zones[0].id
}

# Powering a VM on requires at least one backup SLA policy (provider/API constraint);
# any policy in the VM's availability zone satisfies it.
data "cloudtemple_backup_iaas_opensource_policies" "all" {
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zones.all.availability_zones[0].id
}

# The Ubuntu marketplace image. Catalogs differ between tenants, so the exact name
# is a variable — adjust it (or pass -var) if the apply reports "not found".
data "cloudtemple_marketplace_item" "image" {
  name = var.marketplace_name
}

locals {
  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repositories.all.storage_repositories[0].id
  network_id            = data.cloudtemple_compute_iaas_opensource_networks.all.networks[0].id
  backup_policy_id      = data.cloudtemple_backup_iaas_opensource_policies.all.policies[0].id
}

# --- the VM, deployed from the marketplace ----------------------------------------

resource "cloudtemple_compute_iaas_opensource_virtual_machine" "ubuntu" {
  name        = var.vm_name
  power_state = var.vm_power_state # "on" / "off" — driven by ct-test.sh to exercise start/stop

  marketplace_item_id   = data.cloudtemple_marketplace_item.image.id
  storage_repository_id = local.storage_repository_id

  cpu                  = var.cpu
  memory               = var.memory_gib * 1024 * 1024 * 1024
  num_cores_per_socket = 1
  boot_firmware        = "uefi" # marketplace images deploy UEFI
  secure_boot          = false
  auto_power_on        = false
  high_availability    = "disabled"

  # One adapter on the discovered network (the marketplace item defines how many).
  os_network_adapter {
    network_id = local.network_id
  }

  # The OS disk lives on the discovered storage repository.
  os_disk {
    storage_repository_id = local.storage_repository_id
  }

  # At least one backup policy is mandatory to power the VM on.
  backup_sla_policies = [local.backup_policy_id]
}

# --- an additional data disk, attached to the VM ----------------------------------

resource "cloudtemple_compute_iaas_opensource_virtual_disk" "data" {
  name                  = "${var.vm_name}-data"
  size                  = var.data_disk_gib * 1024 * 1024 * 1024
  mode                  = "RW"
  bootable              = false
  storage_repository_id = local.storage_repository_id
  virtual_machine_id    = cloudtemple_compute_iaas_opensource_virtual_machine.ubuntu.id
}

output "vm_id" {
  value = cloudtemple_compute_iaas_opensource_virtual_machine.ubuntu.id
}

output "vm_power_state" {
  value = cloudtemple_compute_iaas_opensource_virtual_machine.ubuntu.power_state
}

output "data_disk_id" {
  value = cloudtemple_compute_iaas_opensource_virtual_disk.data.id
}
