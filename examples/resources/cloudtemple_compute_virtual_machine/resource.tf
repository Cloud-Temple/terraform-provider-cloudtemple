data "cloudtemple_compute_virtual_datacenter" "TH3S" {
  name = "DC-TH3S"
}

data "cloudtemple_compute_host_cluster" "CLU_001" {
  name = "clu001-ucs12"
}

data "cloudtemple_compute_datastore_cluster" "SDRS_001" {
  name = "sdrs001-LIVE_"
}

# Deploying a new virtual machine with a given operating system
resource "cloudtemple_compute_virtual_machine" "pbt_tf_delete_powered_on" {
  name = "pbt_tf_delete_powered_on"

  memory                 = 8 * 1024 * 1024 * 1024
  cpu                    = 2
  num_cores_per_socket   = 1
  cpu_hot_add_enabled    = true
  cpu_hot_remove_enabled = true
  memory_hot_add_enabled = true
  power_state            = "off"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.TH3S.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.CLU_001.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.SDRS_001.id
  guest_operating_system_moref = "amazonlinux2_64Guest"

  tags = {
    created_by = "Terraform"
  }

  lifecycle {
    prevent_destroy = true
  }
}

# Clone an already existing virtual machine
# resource "cloudtemple_compute_virtual_machine" "cloned" {
#   name = "cloned"

#   clone_virtual_machine_id = cloudtemple_compute_virtual_machine.scratch.id

#   datacenter_id        = data.cloudtemple_compute_virtual_datacenter.dc.id
#   host_cluster_id      = data.cloudtemple_compute_host_cluster.flo.id
#   datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.koukou.id

#   tags = {
#     created_by = "Terraform"
#   }
# }

# Deploy an item from the content library
# data "cloudtemple_compute_content_library" "public" {
#   name = "PUBLIC"
# }

# data "cloudtemple_compute_content_library_item" "centos" {
#   content_library_id = data.cloudtemple_compute_content_library.public.id
#   name               = "20211115132417_master_linux-centos-8"
# }

# data "cloudtemple_compute_datastore" "ds" {
#   name = "ds001-bob-svc1-data4-eqx6"
# }

# resource "cloudtemple_compute_virtual_machine" "content-library" {
#   name = "from-content-library-item"

#   content_library_id      = data.cloudtemple_compute_content_library.public.id
#   content_library_item_id = data.cloudtemple_compute_content_library_item.centos.id

#   datacenter_id        = data.cloudtemple_compute_virtual_datacenter.dc.id
#   host_cluster_id      = data.cloudtemple_compute_host_cluster.flo.id
#   datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.koukou.id
#   datastore_id         = data.cloudtemple_compute_datastore.ds.id

#   deploy_options = {
#     trak_sshpublickey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKpZ5juF5a/CXV9nQ0PANptTG9Gh3J0aj6yVjkF0fSkC remi@cloud-temple.com"
#   }

#   tags = {
#     created_by = "Terraform"
#   }
# }
