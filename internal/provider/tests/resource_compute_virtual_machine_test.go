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
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datacenter_id", os.Getenv(DataCenterId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "host_cluster_id", os.Getenv(HostClusterId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id", os.Getenv(DatastoreClusterId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref", os.Getenv(OperatingSystemMoRef)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.environment", "test"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "triggered_alarms.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineRelocate,
					os.Getenv(DataCenterId),
					os.Getenv(VirtualMachineHostClusterIdRelocate),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datacenter_id", os.Getenv(DataCenterId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "host_cluster_id", os.Getenv(VirtualMachineHostClusterIdRelocate)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id", os.Getenv(DatastoreClusterId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref", os.Getenv(OperatingSystemMoRef)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.environment", "test"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "triggered_alarms.#", "0"),
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
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "memory", "67108864"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "cpu", "2"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "num_cores_per_socket", "2"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "cpu_hot_remove_enabled", "true"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.environment", "demo"),
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
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.%", "0"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachinePowerOn,
					os.Getenv(VmPolicyDaily),
					os.Getenv(VmPolicyWeekly),
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
					os.Getenv(VmPolicyDaily),
					os.Getenv(VmPolicyWeekly),
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
					os.Getenv(VirtualMachineHostClusterIdRelocate),
					os.Getenv(DatastoreClusterId),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "name", "test-terraform-cloned"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "datacenter_id", os.Getenv(DataCenterId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "host_cluster_id", os.Getenv(VirtualMachineHostClusterIdRelocate)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "datastore_cluster_id", os.Getenv(DatastoreClusterId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "tags.environment", "cloned"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineContentLibraryDeploy,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "name", "test-terraform-content-library-deployed"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "datacenter_id", "6ecdc746-3225-489d-be78-2c07f715c8d5"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "host_cluster_id", "bd5d8bf4-953a-46fb-9997-45467ba1ae6f"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "tags.environment", "cloned-from-content-library"),
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
data "cloudtemple_backup_sla_policy" "daily" {
	name = "%s"
}

data "cloudtemple_backup_sla_policy" "weekly" {
	name = "%s"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform-rename"
  power_state = "on"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"

  guest_operating_system_moref = "%s"

  backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.weekly.id,
		data.cloudtemple_backup_sla_policy.daily.id,
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
  name = "local-vc-vstack-001-t0001"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  name               = "test-fsn-ubuntu"
}

data "cloudtemple_compute_datastore" "foo" {
	name = "ds001-t0001-r-stw1-data13-th3s"
}

data "cloudtemple_compute_network" "foo" {
  name = "LAN-dvs-001"
}

resource "cloudtemple_compute_virtual_machine" "content-library-deployed" {
  name = "test-terraform-content-library-deployed"

  content_library_id      = data.cloudtemple_compute_content_library.foo.id
  content_library_item_id = data.cloudtemple_compute_content_library_item.foo.id

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_id          			 = "88fb9089-cf33-47f0-938a-fe792f4a9039"

  tags = {
		"environment" = "cloned-from-content-library"
  }
}
`
