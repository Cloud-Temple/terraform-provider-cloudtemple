package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVirtualMachine,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datacenter_id", "85d53d08-0fa9-491e-ab89-90919516df25"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "host_cluster_id", "dde72065-60f4-4577-836d-6ea074384d62"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id", "6b06b226-ef55-4a0a-92bc-7aa071681b1b"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref", "amazonlinux2_64Guest"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.environment", "test"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "triggered_alarms.#", "0"),
				),
			},
			{
				Config: testAccResourceVirtualMachineRelocate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datacenter_id", "ac33c033-693b-4fc5-9196-26df77291dbb"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "host_cluster_id", "083b0ed7-8b0f-4cec-be47-78f48b457e6a"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id", "1a996110-2746-4725-958f-f6fceef05b32"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref", "amazonlinux2_64Guest"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.environment", "test"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "triggered_alarms.#", "0"),
				),
			},
			{
				Config:            testAccResourceVirtualMachine,
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
				Config: testAccResourceVirtualMachineUpdate,
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
				Config: testAccResourceVirtualMachineRename,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform-rename"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.%", "0"),
				),
			},
			{
				Config: testAccResourceVirtualMachinePowerOn,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform-rename"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "power_state", "on"),
				),
			},
			{
				Destroy: true,
				Config:  testAccResourceVirtualMachinePowerOn,
			},
			{
				Config: testAccResourceVirtualMachineClone,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "name", "test-terraform-cloned"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "datacenter_id", "85d53d08-0fa9-491e-ab89-90919516df25"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "host_cluster_id", "dde72065-60f4-4577-836d-6ea074384d62"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "datastore_cluster_id", "6b06b226-ef55-4a0a-92bc-7aa071681b1b"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.cloned", "tags.environment", "cloned"),
				),
			},
			{
				Config: testAccResourceVirtualMachineContentLibraryDeploy,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "name", "test-terraform-content-library-deployed"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "datacenter_id", "85d53d08-0fa9-491e-ab89-90919516df25"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "host_cluster_id", "dde72065-60f4-4577-836d-6ea074384d62"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.content-library-deployed", "guest_operating_system_moref", "centos8_64Guest"),
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

  datacenter_id                = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id              = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_cluster_id         = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
  guest_operating_system_moref = "amazonlinux2_64Guest"

  tags = {
	"environment" = "test"
  }
}
`

const testAccResourceVirtualMachineRelocate = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  datacenter_id                = "ac33c033-693b-4fc5-9196-26df77291dbb"
  host_cluster_id              = "083b0ed7-8b0f-4cec-be47-78f48b457e6a"
  datastore_cluster_id         = "1a996110-2746-4725-958f-f6fceef05b32"
  guest_operating_system_moref = "amazonlinux2_64Guest"

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

  datacenter_id                = "ac33c033-693b-4fc5-9196-26df77291dbb"
  host_cluster_id              = "083b0ed7-8b0f-4cec-be47-78f48b457e6a"
  datastore_cluster_id         = "1a996110-2746-4725-958f-f6fceef05b32"
  guest_operating_system_moref = "amazonlinux2_64Guest"

  tags = {
	"environment" = "demo"
  }
}
`

const testAccResourceVirtualMachineRename = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-rename"

  datacenter_id                = "ac33c033-693b-4fc5-9196-26df77291dbb"
  host_cluster_id              = "083b0ed7-8b0f-4cec-be47-78f48b457e6a"
  datastore_cluster_id         = "1a996110-2746-4725-958f-f6fceef05b32"
  guest_operating_system_moref = "amazonlinux2_64Guest"

  lifecycle {
	prevent_destroy = true
  }
}
`

const testAccResourceVirtualMachinePowerOn = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform-rename"
  power_state = "on"

  datacenter_id                = "ac33c033-693b-4fc5-9196-26df77291dbb"
  host_cluster_id              = "083b0ed7-8b0f-4cec-be47-78f48b457e6a"
  datastore_cluster_id         = "1a996110-2746-4725-958f-f6fceef05b32"
  guest_operating_system_moref = "amazonlinux2_64Guest"
}
`

const testAccResourceVirtualMachineClone = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  datacenter_id                = "ac33c033-693b-4fc5-9196-26df77291dbb"
  host_cluster_id              = "083b0ed7-8b0f-4cec-be47-78f48b457e6a"
  datastore_cluster_id         = "1a996110-2746-4725-958f-f6fceef05b32"
  guest_operating_system_moref = "amazonlinux2_64Guest"

  tags = {
	"environment" = "test"
  }
}

resource "cloudtemple_compute_virtual_machine" "cloned" {
  name = "test-terraform-cloned"

  clone_virtual_machine_id     = cloudtemple_compute_virtual_machine.foo.id
  datacenter_id                = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id              = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_cluster_id         = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"

  tags = {
	"environment" = "cloned"
  }
}
`

const testAccResourceVirtualMachineContentLibraryDeploy = `
data "cloudtemple_compute_content_library" "foo" {
  name = "PUBLIC"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  name               = "20211115132417_master_linux-centos-8"
}

resource "cloudtemple_compute_virtual_machine" "content-library-deployed" {
  name = "test-terraform-content-library-deployed"

  content_library_id      = data.cloudtemple_compute_content_library.foo.id
  content_library_item_id = data.cloudtemple_compute_content_library_item.foo.id

  datacenter_id         = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id       = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_id          = "d439d467-943a-49f5-a022-c0c25b737022"

  guest_operating_system_moref = "centos8_64Guest"

  deploy_options = {
	trak_sshpublickey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKpZ5juF5a/CXV9nQ0PANptTG9Gh3J0aj6yVjkF0fSkC remi@lenstra.fr"
  }

  tags = {
	"environment" = "cloned-from-content-library"
  }
}
`
