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

# --- networking: put the VM on the LAN, give it a static IP, attach a public IP ---
# The "LAN" exists on two correlated layers: an OpenIaaS network (where the adapter
# connects) and a VPC private network (where a managed static IP is allocated). Both
# are selected BY NAME (var.lan_network_name) — the convention is that they share the
# LAN's name. The public IP is a pre-provisioned floating IP discovered free below.
data "cloudtemple_vpc_private_networks" "all" {}

data "cloudtemple_vpc_floating_ips" "all" {}

locals {
  # Storage repository: pick a USABLE SR (not in maintenance, accessible) with the
  # MOST free space — never the first listed, which may be full/limited (the
  # marketplace deploy then fails "Limited storage capacity"). Mirrors the API
  # cycle's hard filters (maintenance / accessible / capacity); the precondition on
  # the VM fails closed if none has room.
  all_srs    = data.cloudtemple_compute_iaas_opensource_storage_repositories.all.storage_repositories
  usable_srs = [for s in local.all_srs : s if s.id != "" && !s.maintenance_mode && s.accessible != 0]
  # Floor = OS-disk/headroom (var.min_free_gib) + the data disk this config also creates.
  min_free_bytes        = (var.min_free_gib + var.data_disk_gib) * 1024 * 1024 * 1024
  max_free_bytes        = length(local.usable_srs) > 0 ? max([for s in local.usable_srs : s.free_capacity]...) : 0
  storage_repository_id = try([for s in local.usable_srs : s.id if s.free_capacity == local.max_free_bytes][0], null)

  backup_policy_id = data.cloudtemple_backup_iaas_opensource_policies.all.policies[0].id

  # The LAN, by name, on each layer. We keep the full match lists so resource
  # `precondition`s below can assert EXACTLY one match — a 0-match would otherwise be
  # silently null (Terraform's one([]) == null), and a >1-match ambiguous; both must
  # be a clear error, not a wrong silent pick.
  lan_openiaas_matches    = [for n in data.cloudtemple_compute_iaas_opensource_networks.all.networks : n.id if n.name == var.lan_network_name]
  lan_private_matches     = [for p in data.cloudtemple_vpc_private_networks.all.private_networks : p.id if p.name == var.lan_network_name]
  lan_openiaas_network_id = try(local.lan_openiaas_matches[0], null)
  lan_private_network_id  = try(local.lan_private_matches[0], null)

  # FREE public floating IPs (not bound to any static IP). null if none — a
  # precondition on the binding fails visibly rather than picking an in-use address.
  free_floating_ips   = [for f in data.cloudtemple_vpc_floating_ips.all.floating_ips : f.id if f.static_ip_id == ""]
  free_floating_ip_id = try(local.free_floating_ips[0], null)
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

  # One adapter on the LAN (OpenIaaS network selected by name). Its MAC (computed)
  # is what the static IP below binds to.
  os_network_adapter {
    network_id = local.lan_openiaas_network_id
  }

  # The OS disk lives on the discovered storage repository.
  os_disk {
    storage_repository_id = local.storage_repository_id
  }

  # At least one backup policy is mandatory to power the VM on.
  backup_sla_policies = [local.backup_policy_id]

  # Fail closed (clear message) if the LAN name matches zero or several OpenIaaS
  # networks — never silently leave the adapter on a null/wrong network.
  lifecycle {
    precondition {
      condition     = length(local.lan_openiaas_matches) == 1
      error_message = "Expected exactly one OpenIaaS network named \"${var.lan_network_name}\", found ${length(local.lan_openiaas_matches)}. Set var.lan_network_name to your LAN."
    }
    # Fail closed if no usable storage repository has room — never deploy onto a
    # full/limited SR (the API rejects it with "Limited storage capacity").
    precondition {
      condition     = local.max_free_bytes >= local.min_free_bytes
      error_message = "No usable storage repository has at least ${var.min_free_gib + var.data_disk_gib} GiB free (largest usable free = ${floor(local.max_free_bytes / 1024 / 1024 / 1024)} GiB). Free space, or lower var.min_free_gib."
    }
  }
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

# --- a managed static IP on the LAN (VPC private network) -------------------------
# Bound to the VM's adapter by its MAC; ip_address is omitted so the API allocates a
# free, valid address ("une ip qui va bien"). Referencing the VM's MAC makes the
# static IP depend on the VM, so destroy removes the IP association BEFORE the VM —
# no orphaned static IP (the recommended wiring from the resource's own example).
resource "cloudtemple_vpc_static_ip" "lan" {
  private_network_id   = local.lan_private_network_id
  mac_address          = cloudtemple_compute_iaas_opensource_virtual_machine.ubuntu.os_network_adapter[0].mac_address
  resource_description = var.vm_name

  # Fail closed if the LAN name matches zero or several VPC private networks.
  lifecycle {
    precondition {
      condition     = length(local.lan_private_matches) == 1
      error_message = "Expected exactly one VPC private network named \"${var.lan_network_name}\", found ${length(local.lan_private_matches)}. Set var.lan_network_name to your LAN."
    }
  }
}

# --- attach a public (floating) IP to the static IP -------------------------------
# Binds a PRE-PROVISIONED free floating IP (create = bind, destroy = unbind; it never
# creates/destroys the public IP). free_floating_ip_id is null if none is free, which
# fails the apply visibly rather than touching an in-use address.
resource "cloudtemple_vpc_floating_ip_binding" "public" {
  floating_ip_id = local.free_floating_ip_id
  static_ip_id   = cloudtemple_vpc_static_ip.lan.id

  # Fail closed (clear message) if no public IP is free, rather than passing null.
  lifecycle {
    precondition {
      condition     = local.free_floating_ip_id != null
      error_message = "No free public (floating) IP available — all are bound. Free one or provision a new public IP in the VPC."
    }
  }
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

output "lan_static_ip" {
  description = "The static IP allocated to the VM on the LAN."
  value       = cloudtemple_vpc_static_ip.lan.ip_address
}

output "public_ip" {
  description = "The public (floating) IP bound to the VM, for external reachability tests."
  value       = cloudtemple_vpc_floating_ip_binding.public.floating_ip_address
}
