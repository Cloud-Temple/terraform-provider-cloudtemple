package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceVirtualDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVirtualDisk,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "provisioning_type", "dynamic"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_mode", "persistent"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "capacity", "10737418240"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_id", "0eb76e31-5214-41d5-a834-803733a8dbb2"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "name", "Hard disk 1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "machine_manager_id", "8afdb4e8-b68d-4bb8-a606-3dc47cc2da0e"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_unit_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "controller_bus_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_name", "ds003-t0001-r-stw1-data13-th3s"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "instant_access", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "native_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "editable", "true"),
				),
			},
			{
				Config: testAccResourceVirtualDiskResize,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "provisioning_type", "dynamic"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_mode", "persistent"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "capacity", "21474836480"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_id", "0eb76e31-5214-41d5-a834-803733a8dbb2"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "name", "Hard disk 1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "machine_manager_id", "8afdb4e8-b68d-4bb8-a606-3dc47cc2da0e"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_unit_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "controller_bus_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_name", "ds003-t0001-r-stw1-data13-th3s"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "instant_access", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "native_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "editable", "true"),
				),
			},
		},
	})
}

const testAccResourceVirtualDisk = `
data "cloudtemple_compute_machine_manager" "vstack-001" {
  name = "vc-vstack-001-t0001"
}

data "cloudtemple_compute_virtual_datacenter" "TH3S" {
  name = "DC-TH3S"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

data "cloudtemple_compute_host_cluster" "CLU001" {
  name               = "clu001-ucs12"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

data "cloudtemple_compute_datastore" "DS003" {
  name = "ds003-t0001-r-stw1-data13-th3s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-virtual-disk"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.TH3S.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.CLU001.id
  datastore_id         				 = data.cloudtemple_compute_datastore.DS003.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

resource "cloudtemple_compute_virtual_disk" "foo" {
  virtual_machine_id   = cloudtemple_compute_virtual_machine.foo.id
  provisioning_type    = "dynamic"
  disk_mode            = "persistent"
  capacity             = 10737418240
  datastore_id         				 = data.cloudtemple_compute_datastore.DS003.id
}
`

const testAccResourceVirtualDiskResize = `
data "cloudtemple_compute_machine_manager" "vstack-001" {
  name = "vc-vstack-001-t0001"
}

data "cloudtemple_compute_virtual_datacenter" "TH3S" {
  name = "DC-TH3S"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

data "cloudtemple_compute_host_cluster" "CLU001" {
  name               = "clu001-ucs12"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

data "cloudtemple_compute_datastore" "DS003" {
  name = "ds003-t0001-r-stw1-data13-th3s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-001.id
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-virtual-disk"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.TH3S.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.CLU001.id
  datastore_id         				 = data.cloudtemple_compute_datastore.DS003.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

resource "cloudtemple_compute_virtual_disk" "foo" {
  virtual_machine_id   = cloudtemple_compute_virtual_machine.foo.id
  provisioning_type    = "dynamic"
  disk_mode            = "persistent"
  capacity             = 2 * 10737418240
  datastore_id = data.cloudtemple_compute_datastore.DS003.id
}
`
