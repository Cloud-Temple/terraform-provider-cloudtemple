package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceNetworkAdapter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceNetworkAdapter,
					os.Getenv(MachineManagerName),
					os.Getenv(VirtualDatacenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DatastoreClusterName),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", os.Getenv(NetworkAdapterName)),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceNetworkAdapterAssigned,
					os.Getenv(MachineManagerName),
					os.Getenv(VirtualDatacenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DatastoreClusterName),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", os.Getenv(NetworkAdapterName)),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceNetworkAdapterConnected,
					os.Getenv(MachineManagerName),
					os.Getenv(VirtualDatacenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DatastoreClusterName),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "true"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", os.Getenv(NetworkAdapterName)),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceNetworkAdapterDisconnected,
					os.Getenv(MachineManagerName),
					os.Getenv(VirtualDatacenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DatastoreClusterName),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", os.Getenv(NetworkAdapterName)),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceNetworkAdapterPowerOff,
					os.Getenv(MachineManagerName),
					os.Getenv(VirtualDatacenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DatastoreClusterName),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", os.Getenv(NetworkAdapterName)),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceNetworkAdapter,
					os.Getenv(MachineManagerName),
					os.Getenv(VirtualDatacenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DatastoreClusterName),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
				),
				ResourceName:      "cloudtemple_compute_network_adapter.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccResourceNetworkAdapter = `
data "cloudtemple_compute_machine_manager" "vstack" {
	name = "%s"
}

data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_host_cluster" "chc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_datastore_cluster" "cdc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_backup_sla_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-network-adapter"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"

  backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.nobackup.id,
  ]
}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}


resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.foo.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
  mac_address        = "00:50:56:86:4a:27"
}
`

const testAccResourceNetworkAdapterAssigned = `
data "cloudtemple_compute_machine_manager" "vstack" {
	name = "%s"
}

data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_host_cluster" "chc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_datastore_cluster" "cdc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name = "test-terraform-network-adapter"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"
}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
}
`

const testAccResourceNetworkAdapterConnected = `
data "cloudtemple_compute_machine_manager" "vstack" {
	name = "%s"
}

data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_host_cluster" "chc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_datastore_cluster" "cdc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_backup_sla_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"
  power_state = "on"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"

  backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.nobackup.id
  ]

}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  connected          = true
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
}
`

const testAccResourceNetworkAdapterDisconnected = `
data "cloudtemple_compute_machine_manager" "vstack" {
	name = "%s"
}

data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_host_cluster" "chc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_datastore_cluster" "cdc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_backup_sla_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"
  power_state = "on"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"

  backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.nobackup.id
  ]
}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
}
`

const testAccResourceNetworkAdapterPowerOff = `
data "cloudtemple_compute_machine_manager" "vstack" {
	name = "%s"
}

data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_host_cluster" "chc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_datastore_cluster" "cdc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_backup_sla_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"

  backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.nobackup.id
  ]
}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
}
`
