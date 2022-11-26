data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
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

  virtual_datacenter_id        = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"


  tags = {
    created_by = "Terraform"
  }
}

# Clone an already existing virtual machine
resource "cloudtemple_compute_virtual_machine" "cloned" {
  name = "cloned"

  clone_virtual_machine_id = cloudtemple_compute_virtual_machine.scratch.id

  virtual_datacenter_id = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id       = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id  = data.cloudtemple_compute_datastore_cluster.koukou.id

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

resource "cloudtemple_compute_virtual_machine" "content-library" {
  name = "from-content-library-item"

  content_library_id      = data.cloudtemple_compute_content_library.public.id
  content_library_item_id = data.cloudtemple_compute_content_library_item.centos.id

  virtual_datacenter_id = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id       = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id  = data.cloudtemple_compute_datastore_cluster.koukou.id
  datastore_id          = data.cloudtemple_compute_datastore.ds.id

  tags = {
    created_by = "Terraform"
  }
}
