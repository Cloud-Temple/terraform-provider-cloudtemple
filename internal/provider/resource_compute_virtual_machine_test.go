package provider

import (
	"regexp"
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
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "virtual_datacenter_id", "85d53d08-0fa9-491e-ab89-90919516df25"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "host_cluster_id", "dde72065-60f4-4577-836d-6ea074384d62"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id", "6b06b226-ef55-4a0a-92bc-7aa071681b1b"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref", "amazonlinux2_64Guest"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "tags.environment", "test"),
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
				// Trying to destroy a running VM will not work
				Destroy:     true,
				Config:      testAccResourceVirtualMachinePowerOn,
				ExpectError: regexp.MustCompile(`NOT_ALLOWED_IN_CURRENT_STATE`),
			},
			{
				// We stop the VM so that we can destroy it
				Config: testAccResourceVirtualMachine,
			},
		},
	})
}

const testAccResourceVirtualMachine = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  virtual_datacenter_id        = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id              = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_cluster_id         = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
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

  virtual_datacenter_id        = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id              = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_cluster_id         = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
  guest_operating_system_moref = "amazonlinux2_64Guest"

  tags = {
	"environment" = "demo"
  }
}
`

const testAccResourceVirtualMachineRename = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-rename"

  virtual_datacenter_id        = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id              = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_cluster_id         = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
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

  virtual_datacenter_id        = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id              = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_cluster_id         = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
  guest_operating_system_moref = "amazonlinux2_64Guest"
}
`
