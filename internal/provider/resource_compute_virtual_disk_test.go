package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	DataCenterName = "DATACENTER_NAME"
	DataStoreId2   = "COMPUTE_DATASTORE_ID_2"
	DataStoreName2 = "COMPUTE_DATASTORE_NAME_2"
)

func TestAccResourceVirtualDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualDisk,
					os.Getenv(MachineManagerName),
					os.Getenv(DataCenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DataStoreName2),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "provisioning_type", "dynamic"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_mode", "persistent"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "capacity", "10737418240"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_id", os.Getenv(DataStoreId2)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "name", os.Getenv(VirtualDiskName)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "machine_manager_id", os.Getenv(MachineManagerId2)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_unit_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "controller_bus_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_name", os.Getenv(DataStoreName2)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "instant_access", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "native_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "editable", "true"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualDiskResize,
					os.Getenv(MachineManagerName),
					os.Getenv(DataCenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DataStoreName2),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "provisioning_type", "dynamic"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_mode", "persistent"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "capacity", "21474836480"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_id", os.Getenv(DataStoreId2)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "name", os.Getenv(VirtualDiskName)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "machine_manager_id", os.Getenv(MachineManagerId2)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "disk_unit_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "controller_bus_number", "0"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "datastore_name", os.Getenv(DataStoreName2)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "instant_access", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_disk.foo", "native_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_disk.foo", "editable", "true"),
				),
			},
		},
	})
}

const testAccResourceVirtualDisk = `
data "cloudtemple_compute_machine_manager" "vstack" {
  name = "%s"
}

data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_host_cluster" "hc" {
  name               = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_datastore" "ds" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-virtual-disk"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.hc.id
  datastore_id         		   = data.cloudtemple_compute_datastore.ds.id
  guest_operating_system_moref = "%s"
}

resource "cloudtemple_compute_virtual_disk" "foo" {
  virtual_machine_id   = cloudtemple_compute_virtual_machine.foo.id
  provisioning_type    = "dynamic"
  disk_mode            = "persistent"
  capacity             = 10737418240
  datastore_id         = data.cloudtemple_compute_datastore.ds.id
}
`

const testAccResourceVirtualDiskResize = `
data "cloudtemple_compute_machine_manager" "vstack" {
  name = "%s"
}

data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_host_cluster" "hc" {
  name               = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_datastore" "ds" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-virtual-disk"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.hc.id
  datastore_id         		   = data.cloudtemple_compute_datastore.ds.id
  guest_operating_system_moref = "%s"
}

resource "cloudtemple_compute_virtual_disk" "foo" {
  virtual_machine_id   = cloudtemple_compute_virtual_machine.foo.id
  provisioning_type    = "dynamic"
  disk_mode            = "persistent"
  capacity             = 2 * 10737418240
  datastore_id = data.cloudtemple_compute_datastore.ds.id
}
`
