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
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_id", "d439d467-943a-49f5-a022-c0c25b737022"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_cluster_id", "6b06b226-ef55-4a0a-92bc-7aa071681b1b"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "name", "Hard disk 1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "machine_manager_id", "9dba240e-a605-4103-bac7-5336d3ffd124"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_unit_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "controller_bus_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_name", "ds001-bob-svc1-data4-eqx6"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "instant_access", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "native_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_path", "[ds001-bob-svc1-data4-eqx6] test-terraform-network-adapter_1/test-terraform-network-adapter.vmdk"),
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
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_id", "d439d467-943a-49f5-a022-c0c25b737022"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_cluster_id", "6b06b226-ef55-4a0a-92bc-7aa071681b1b"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "name", "Hard disk 1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "machine_manager_id", "9dba240e-a605-4103-bac7-5336d3ffd124"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_unit_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "controller_bus_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_name", "ds001-bob-svc1-data4-eqx6"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "instant_access", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "native_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_path", "[ds001-bob-svc1-data4-eqx6] test-terraform-network-adapter_1/test-terraform-network-adapter.vmdk"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "editable", "true"),
				),
			},
		},
	})
}

const testAccResourceVirtualDisk = `
data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-network-adapter"

  virtual_datacenter_id        = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

resource "cloudtemple_compute_virtual_disk" "foo" {
  virtual_machine_id   = cloudtemple_compute_virtual_machine.foo.id
  provisioning_type    = "dynamic"
  disk_mode            = "persistent"
  capacity             = 10737418240
  datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.koukou.id
}
`

const testAccResourceVirtualDiskResize = `
data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-network-adapter"

  virtual_datacenter_id        = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

resource "cloudtemple_compute_virtual_disk" "foo" {
  virtual_machine_id   = cloudtemple_compute_virtual_machine.foo.id
  provisioning_type    = "dynamic"
  disk_mode            = "persistent"
  capacity             = 2 * 10737418240
  datastore_cluster_id = data.cloudtemple_compute_datastore_cluster.koukou.id
}
`
