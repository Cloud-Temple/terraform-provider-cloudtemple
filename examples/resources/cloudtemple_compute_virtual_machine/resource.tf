data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

data "cloudtemple_backup_sla_policy" "sla001-daily-par7s" {
  name = "sla001-daily-par7s"
}

data "cloudtemple_backup_sla_policy" "sla001-weekly-par7s" {
  name = "sla001-weekly-par7s"
}

# Deploying a new virtual machine with a given operating system
resource "cloudtemple_compute_virtual_machine" "scratch" {
  name = "from-scratch"

  memory                 = 8 * 1024 * 1024 * 1024
  cpu                    = 2
  num_cores_per_socket   = 1
  cpu_hot_add_enabled    = true
  cpu_hot_remove_enabled = true
  memory_hot_add_enabled = true

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"

  tags = {
    created_by = "Terraform"
  }

  backup_sla_policies = [
    data.cloudtemple_backup_sla_policy.sla001-daily-par7s.id,
    data.cloudtemple_backup_sla_policy.sla001-weekly-par7s.id,
  ]
}

# Clone an already existing virtual machine
resource "cloudtemple_compute_virtual_machine" "cloned" {
  name = "cloned"

  clone_virtual_machine_id = cloudtemple_compute_virtual_machine.scratch.id

  datacenter_id        = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id      = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.koukou.id

  tags = {
    created_by = "Terraform"
  }
}

# Deploy an item from the content library
data "cloudtemple_compute_content_library" "public" {
  name = "PUBLIC"
}

data "cloudtemple_compute_content_library_item" "centos" {
  content_library_id = data.cloudtemple_compute_content_library.public.id
  name               = "20211115132417_master_linux-centos-8"
}

data "cloudtemple_compute_datastore" "ds" {
  name = "ds001-bob-svc1-data4-eqx6"
}

data "cloudtemple_compute_network" "vlan" {
  name = "VLAN_201"
}

resource "cloudtemple_compute_virtual_machine" "content-library" {
  name = "from-content-library-item"

  content_library_id      = data.cloudtemple_compute_content_library.public.id
  content_library_item_id = data.cloudtemple_compute_content_library_item.centos.id

  datacenter_id        = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id      = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.koukou.id
  datastore_id         = data.cloudtemple_compute_datastore.ds.id

  os_disk {
    capacity = 25 * 1024 * 1024 * 1024
  }

  os_network_adapter {
    network_id = data.cloudtemple_compute_network.vlan.id
  }

  tags = {
    created_by = "Terraform"
  }
}

# Deploy an OVF from the content library and configure it with cloud-init
data "cloudtemple_compute_machine_manager" "vstack-001" {
  name = "vc-vstack-001-t0001"
}

data "cloudtemple_compute_virtual_datacenter" "TH3S" {
  name               = "DC-TH3S"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

data "cloudtemple_compute_host_cluster" "CLU001" {
  name               = "clu001-ucs12"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

data "cloudtemple_compute_datastore" "DS003" {
  name               = "ds003-t0001-r-stw1-data13-th3s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

data "cloudtemple_backup_sla_policy" "sla001-daily-par7s" {
  name = "sla001-daily-par7s"
}

data "cloudtemple_backup_sla_policy" "sla001-weekly-par7s" {
  name = "sla001-weekly-par7s"
}

data "cloudtemple_compute_content_library" "local" {
  name = "local-vc-vstack-001-t0001"
}

data "cloudtemple_compute_content_library_item" "ubuntu-cloudimg" {
  content_library_id = data.cloudtemple_compute_content_library.local.id
  name               = "ubuntu-jammy-22.04-cloudimg"
}

resource "cloudtemple_compute_virtual_machine" "ubuntu-cloud-init" {
  name = "ubuntu-cloud-init"

  memory                 = 8 * 1024 * 1024 * 1024
  cpu                    = 2
  num_cores_per_socket   = 1
  cpu_hot_add_enabled    = true
  cpu_hot_remove_enabled = true
  memory_hot_add_enabled = true

  datacenter_id   = data.cloudtemple_compute_virtual_datacenter.TH3S.id
  host_cluster_id = data.cloudtemple_compute_host_cluster.CLU001.id
  datastore_id    = data.cloudtemple_compute_datastore.DS003.id

  content_library_id      = data.cloudtemple_compute_content_library.local.id
  content_library_item_id = data.cloudtemple_compute_content_library_item.ubuntu-cloudimg.id

  power_state = "on"

  backup_sla_policies = [
    data.cloudtemple_backup_sla_policy.sla001-daily-par7s.id,
    data.cloudtemple_backup_sla_policy.sla001-weekly-par7s.id,
  ]

  cloud_init = {
    network-config = filebase64("./cloud-init/network-config.yml")
    user-data      = filebase64("./cloud-init/user-data.yml")
  }
}

# Customize Guest OS using the customize block (deployed from an ubuntu OVF)
resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "pbt-customize-tf-04"
  power_state = "on"

  memory               = 8 * 1024 * 1024 * 1024
  cpu                  = 4
  num_cores_per_socket = 1

  datacenter_id        = data.cloudtemple_compute_virtual_datacenter.th3s.id
  host_cluster_id      = data.cloudtemple_compute_host_cluster.clu001.id
  datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.sdrs001.id
  datastore_id         = data.cloudtemple_compute_datastore.ds001.id

  content_library_id      = data.cloudtemple_compute_content_library.cl001.id
  content_library_item_id = data.cloudtemple_compute_content_library_item.ubuntu.id

  backup_sla_policies = [
    data.cloudtemple_backup_sla_policy.nobackup.id
  ]

  os_network_adapter {
    network_id   = data.cloudtemple_compute_network.lan1.id
    auto_connect = true
  }

  customize {
    network_config {
      hostname = "Pimp-My-VM-updated"
      domain   = "cloud-temple.com"
      dns_server_list = [
        "8.8.8.8"
      ]
      adapters {
        ip_address  = "192.168.16.15"
        subnet_mask = "255.255.255.0"
        gateway     = "192.168.16.254"
      }
    }
  }
}

# Customize Guest OS using the customize block (Deployed from a Windows Server Clone)
resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "pbt-customize-tf-03"
  power_state = "on"

  memory               = 8 * 1024 * 1024 * 1024
  cpu                  = 4
  num_cores_per_socket = 1

  datacenter_id        = data.cloudtemple_compute_virtual_datacenter.th3s.id
  host_cluster_id      = data.cloudtemple_compute_host_cluster.clu001.id
  datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.sdrs001.id
  datastore_id         = data.cloudtemple_compute_datastore.ds001.id

  clone_virtual_machine_id = data.cloudtemple_compute_virtual_machine.windows_server_vm.id

  backup_sla_policies = [
    data.cloudtemple_backup_sla_policy.nobackup.id
  ]

  os_network_adapter {
    network_id   = data.cloudtemple_compute_network.lan1.id
    auto_connect = true
  }

  customize {
    network_config {
      hostname = "Pimp-My-VM"
      domain   = "cloud-temple.com"
      dns_server_list = [
        "8.8.8.8"
      ]
      adapters {
        ip_address  = "192.168.16.12"
        subnet_mask = "255.255.255.0"
        gateway     = "192.168.16.254"
      }
    }
    windows_config {
      auto_logon       = true
      auto_logon_count = 3
      timezone         = 105
      password         = "azerty"
      workgroup        = "cloudtemple"
    }
  }
}