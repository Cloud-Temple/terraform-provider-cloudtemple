# Exemple d'utilisation des extra_config pour une machine virtuelle
# avec les configurations spécifiques pour Ignition et CoreOS

resource "cloudtemple_compute_virtual_machine" "coreos_vm" {
  name                = "coreos-ignition-vm"
  datacenter_id       = var.datacenter_id
  host_cluster_id     = var.host_cluster_id
  datastore_id        = var.datastore_id
  
  # Configuration de base
  memory = 2147483648  # 2GB en bytes
  cpu    = 2
  
  # Système d'exploitation
  guest_operating_system_moref = var.coreos_guest_os_moref
  
  # État d'alimentation
  power_state = "on"
  
  # Configurations supplémentaires (extra_config)
  # Toutes les configurations disponibles pour les machines virtuelles
  extra_config = {
    # === Configuration Ignition (CoreOS) ===
    # Configuration Ignition (données encodées en base64)
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
          name = "docker.service"
          enabled = true
        }]
      }
    }))
    
    # Encodage de la configuration Ignition (seule valeur supportée : "base64")
    "guestinfo.ignition.config.data.encoding" = "base64"
    
    # Configuration réseau pour Afterburn (CoreOS)
    "guestinfo.afterburn.initrd.network-kargs" = "ip=dhcp"
    
    # === Configuration Performance ===
    # Activation du steal clock pour les environnements virtualisés
    "stealclock.enable" = "true"
    
    # === Configuration Disque ===
    # Activation des UUID pour les disques (recommandé pour la plupart des OS)
    "disk.enableUUID" = "true"
    
    # === Configuration PCI Passthrough ===
    # Activation du MMIO 64-bit pour PCI Passthrough
    "pciPassthru.use64BitMMIO" = "true"
    
    # Taille du MMIO 64-bit en GB (ajuster selon les besoins)
    "pciPassthru.64bitMMioSizeGB" = "64"
  }
  
  # Tags pour l'organisation
  tags = {
    Environment = "production"
    OS          = "CoreOS"
    Purpose     = "container-host"
  }
}

# Variables nécessaires
variable "datacenter_id" {
  description = "ID du datacenter"
  type        = string
}

variable "host_cluster_id" {
  description = "ID du cluster d'hôtes"
  type        = string
}

variable "datastore_id" {
  description = "ID du datastore"
  type        = string
}

variable "coreos_guest_os_moref" {
  description = "Moref du système d'exploitation CoreOS"
  type        = string
}

# Outputs pour référence
output "vm_id" {
  description = "ID de la machine virtuelle créée"
  value       = cloudtemple_compute_virtual_machine.coreos_vm.id
}

output "vm_moref" {
  description = "Moref de la machine virtuelle"
  value       = cloudtemple_compute_virtual_machine.coreos_vm.moref
}

output "extra_config_applied" {
  description = "Configurations supplémentaires appliquées"
  value       = cloudtemple_compute_virtual_machine.coreos_vm.extra_config
}
