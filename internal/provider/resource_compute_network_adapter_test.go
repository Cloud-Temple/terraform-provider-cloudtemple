package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceNetworkAdapter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNetworkAdapter,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", "5e029210-b433-4c45-93be-092cef684edc"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", "MANUAL"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", "Network adapter 1"),
				),
			},
			{
				Config: testAccResourceNetworkAdapterAssigned,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", "5e029210-b433-4c45-93be-092cef684edc"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", "ASSIGNED"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", "Network adapter 1"),
				),
			},
			{
				Config: testAccResourceNetworkAdapterConnected,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", "5e029210-b433-4c45-93be-092cef684edc"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", "ASSIGNED"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "true"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", "Network adapter 1"),
				),
			},
			{
				Config: testAccResourceNetworkAdapterDisconnected,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", "5e029210-b433-4c45-93be-092cef684edc"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", "ASSIGNED"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", "Network adapter 1"),
				),
			},
			{
				Config: testAccResourceNetworkAdapterPowerOff,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", "5e029210-b433-4c45-93be-092cef684edc"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", "VMXNET3"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", "ASSIGNED"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "auto_connect", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "connected", "false"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "name", "Network adapter 1"),
				),
			},
			{
				Config:            testAccResourceNetworkAdapter,
				ResourceName:      "cloudtemple_compute_network_adapter.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccResourceNetworkAdapter = `
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

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

data "cloudtemple_compute_network" "foo" {
  name = "VLAN_201"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.foo.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
  mac_address        = "00:50:57:CA:89:B8"
}
`

const testAccResourceNetworkAdapterAssigned = `
data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name = "test-terraform-network-adapter"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

data "cloudtemple_compute_network" "foo" {
  name = "VLAN_201"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
  mac_type           = "ASSIGNED"
}
`

const testAccResourceNetworkAdapterConnected = `
data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"
  power_state = "on"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

data "cloudtemple_compute_network" "foo" {
  name = "VLAN_201"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  connected          = true
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
  mac_type           = "ASSIGNED"
}
`

const testAccResourceNetworkAdapterDisconnected = `
data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"
  power_state = "on"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

data "cloudtemple_compute_network" "foo" {
  name = "VLAN_201"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
  mac_type           = "ASSIGNED"
}
`

const testAccResourceNetworkAdapterPowerOff = `
data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

data "cloudtemple_compute_network" "foo" {
  name = "VLAN_201"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "VMXNET3"
  mac_type           = "ASSIGNED"
}
`
