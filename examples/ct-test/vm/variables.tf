variable "vm_name" {
  description = "Name of the VM (and prefix of its data disk). ct-test.sh injects a run-unique value to avoid colliding with a prior orphan."
  type        = string
  default     = "ct-validate-tf-ubuntu"
}

variable "vm_power_state" {
  description = "VM power state — \"on\" or \"off\". ct-test.sh flips this to exercise stop/start, with a convergence check at each state."
  type        = string
  default     = "on"

  validation {
    condition     = contains(["on", "off"], var.vm_power_state)
    error_message = "vm_power_state must be \"on\" or \"off\"."
  }
}

variable "lan_network_name" {
  description = "Name of the LAN — matched on BOTH the OpenIaaS network (the adapter connects here) and the VPC private network (the static IP is allocated here), which share this name by convention. Adjust to your tenant's LAN."
  type        = string
  default     = "LAN"
}

variable "marketplace_name" {
  description = "Exact name of the Ubuntu marketplace image. Catalogs differ between tenants — adjust if the apply reports the item is not found."
  type        = string
  default     = "Ubuntu 24.04 LTS"
}

variable "cpu" {
  description = "vCPUs for the VM. Raise it if the marketplace image requires more (the order layer rejects a deploy below the image's own minimum)."
  type        = number
  default     = 2
}

variable "memory_gib" {
  description = "Memory in GiB. Raise it if the marketplace image requires more (MEMORY_CONSTRAINT_VIOLATION_ORDER on a too-small deploy)."
  type        = number
  default     = 4
}

variable "data_disk_gib" {
  description = "Size of the additional data disk, in GiB."
  type        = number
  default     = 1
}

variable "min_free_gib" {
  description = "Free-capacity floor (GiB) for the OS disk + headroom; the data disk size (var.data_disk_gib) is added automatically to form the real floor. Among usable SRs (not in maintenance, accessible) the one with the most free space is chosen; the apply fails closed if none reaches the floor."
  type        = number
  default     = 20

  validation {
    condition     = var.min_free_gib > 0
    error_message = "min_free_gib must be > 0."
  }
}
