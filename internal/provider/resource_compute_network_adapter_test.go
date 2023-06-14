package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	NetworkAdapterType           = "COMPUTE_NETWORK_ADAPTER_TYPE"
	NetworkAdapterMacAddress     = "COMPUTE_NETWORK_ADAPTER_MAC_ADDRESS"
	NetworkAdapterMacAddressType = "COMPUTE_NETWORK_ADAPTER_MAC_ADDRESS_TYPE"
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
					os.Getenv(VmPolicyDaily),
					os.Getenv(VmPolicyWeekly),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
					os.Getenv(NetworkAdapterType),
					os.Getenv(NetworkAdapterMacAddress),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", os.Getenv(NetworkAdapterType)),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", "MANUAL"),
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
					os.Getenv(NetworkAdapterType),
					os.Getenv(NetworkAdapterMacAddressType),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", os.Getenv(NetworkAdapterType)),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", os.Getenv(NetworkAdapterMacAddressType)),
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
					os.Getenv(VmPolicyDaily),
					os.Getenv(VmPolicyWeekly),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
					os.Getenv(NetworkAdapterType),
					os.Getenv(NetworkAdapterMacAddressType),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", os.Getenv(NetworkAdapterType)),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", os.Getenv(NetworkAdapterMacAddressType)),
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
					os.Getenv(VmPolicyDaily),
					os.Getenv(VmPolicyWeekly),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
					os.Getenv(NetworkAdapterType),
					os.Getenv(NetworkAdapterMacAddressType),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", os.Getenv(NetworkAdapterType)),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", os.Getenv(NetworkAdapterMacAddressType)),
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
					os.Getenv(VmPolicyDaily),
					os.Getenv(VmPolicyWeekly),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
					os.Getenv(NetworkAdapterType),
					os.Getenv(NetworkAdapterMacAddressType),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "network_id", os.Getenv(NetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "type", os.Getenv(NetworkAdapterType)),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_network_adapter.foo", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_network_adapter.foo", "mac_type", os.Getenv(NetworkAdapterMacAddressType)),
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
					os.Getenv(VmPolicyDaily),
					os.Getenv(VmPolicyWeekly),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(NetworkName),
					os.Getenv(NetworkAdapterType),
					os.Getenv(NetworkAdapterMacAddress),
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

data "cloudtemple_backup_sla_policy" "daily" {
	name = "%s"
}

data "cloudtemple_backup_sla_policy" "weekly" {
	name = "%s"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-network-adapter"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"

  backup_sla_policies = [
	data.cloudtemple_backup_sla_policy.weekly.id,
	data.cloudtemple_backup_sla_policy.daily.id,
  ]
}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}


resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.foo.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "%s"
  mac_address        = "%s"
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
  type               = "%s"
  mac_type           = "%s"
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

data "cloudtemple_backup_sla_policy" "daily" {
	name = "%s"
}

data "cloudtemple_backup_sla_policy" "weekly" {
	name = "%s"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"
  power_state = "on"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"

  backup_sla_policies = [
	data.cloudtemple_backup_sla_policy.weekly.id,
	data.cloudtemple_backup_sla_policy.daily.id,
  ]

}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  connected          = true
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "%s"
  mac_type           = "%s"
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

data "cloudtemple_backup_sla_policy" "daily" {
	name = "%s"
}

data "cloudtemple_backup_sla_policy" "weekly" {
	name = "%s"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"
  power_state = "on"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"

  backup_sla_policies = [
	data.cloudtemple_backup_sla_policy.weekly.id,
	data.cloudtemple_backup_sla_policy.daily.id,
  ]
}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "%s"
  mac_type           = "%s"
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

data "cloudtemple_backup_sla_policy" "daily" {
	name = "%s"
}

data "cloudtemple_backup_sla_policy" "weekly" {
	name = "%s"
}

resource "cloudtemple_compute_virtual_machine" "bar" {
  name        = "test-terraform-network-adapter-connected"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"

  backup_sla_policies = [
	data.cloudtemple_backup_sla_policy.weekly.id,
	data.cloudtemple_backup_sla_policy.daily.id,
  ]
}

data "cloudtemple_compute_network" "foo" {
  name = "%s"
}

resource "cloudtemple_compute_network_adapter" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.bar.id
  network_id         = data.cloudtemple_compute_network.foo.id
  type               = "%s"
  mac_type           = "%s"
}
`
