package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	DataCenterId                        = "DATACENTER_ID"
	VirtualMachineHostClusterIdRelocate = "COMPUTE_VIRTUAL_MACHINE_HOST_CLUSTER_RELOCATE"
	VmPolicyDaily                       = "COMPUTE_VIRTUAL_MACHINE_POLICY_1"
	VmPolicyWeekly                      = "COMPUTE_VIRTUAL_MACHINE_POLICY_2"
)

func TestAccResourceVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachine,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "datacenter_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "host_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "tags.environment"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "triggered_alarms.#"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineRelocate,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "datacenter_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "host_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "tags.environment"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "triggered_alarms.#"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachine,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				ResourceName:      "cloudtemple_compute_virtual_machine.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"datastore_cluster_id",
					"guest_operating_system_moref",
					"host_cluster_id",
					"extra_config",
				},
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineUpdate,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "memory"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "cpu"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "num_cores_per_socket"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "cpu_hot_add_enabled"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "cpu_hot_remove_enabled"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "memory_hot_add_enabled"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "tags.environment"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineRename,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform-rename"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachinePowerOn,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform-rename"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "power_state", "on"),
				),
			},
			{
				Destroy: true,
				Config: fmt.Sprintf(
					testAccResourceVirtualMachinePowerOn,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineClone,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "datacenter_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "host_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "datastore_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "tags.environment"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineContentLibraryDeploy,
					os.Getenv(ContentLibraryName),
					os.Getenv(DataStoreName),
					os.Getenv(MachineManagerId),
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.content-library-deployed", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.content-library-deployed", "datacenter_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.content-library-deployed", "host_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.content-library-deployed", "tags.environment"),
				),
			},
		},
	})
}

const testAccResourceVirtualMachine = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  tags = {
		"environment" = "test"
  }
}
`

const testAccResourceVirtualMachineRelocate = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  tags = {
    "environment" = "test"
  }
}
`

const testAccResourceVirtualMachineUpdate = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  memory                 = 2 * 33554432
  cpu                    = 2
  num_cores_per_socket   = 2
  cpu_hot_add_enabled    = true
  cpu_hot_remove_enabled = true
  memory_hot_add_enabled = true

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  tags = {
		"environment" = "demo"
  }
}
`

const testAccResourceVirtualMachineRename = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-rename"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  lifecycle {
		prevent_destroy = true
  }
}
`

const testAccResourceVirtualMachinePowerOn = `
data "cloudtemple_backup_sla_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform-rename"
  power_state = "on"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"

  guest_operating_system_moref = "%s"

  backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.nobackup.id
  ]
}
`

const testAccResourceVirtualMachineClone = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  tags = {
		"environment" = "test"
  }
}

resource "cloudtemple_compute_virtual_machine" "cloned" {
  name = "test-terraform-cloned"

  clone_virtual_machine_id     = cloudtemple_compute_virtual_machine.foo.id
  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"

  tags = {
		"environment" = "cloned"
  }
}
`

const testAccResourceVirtualMachineContentLibraryDeploy = `
data "cloudtemple_compute_content_library" "foo" {
  name = "%s"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  name               = "ubuntu-jammy-22.04-cloudimg"
}

data "cloudtemple_compute_datastore" "foo" {
	name = "%s"
	machine_manager_id = "%s"
}

resource "cloudtemple_compute_virtual_machine" "content-library-deployed" {
  name = "test-terraform-content-library-deployed"

  content_library_id      = data.cloudtemple_compute_content_library.foo.id
  content_library_item_id = data.cloudtemple_compute_content_library_item.foo.id

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_id          			 = data.cloudtemple_compute_datastore.foo.id

  tags = {
		"environment" = "deployed-from-content-library"
  }
}
`
