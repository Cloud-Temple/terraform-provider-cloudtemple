package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceIaasOpensourceVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachine,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "name", "test-terraform-iaas-opensource-vm"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "cpu", "2"),
					//resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "memory", "2147483648"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "pool_id"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachineUpdate,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "name", "test-terraform-iaas-opensource-vm-updated"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "cpu", "4"),
					//resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "memory", "4294967296"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "pool_id"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachine,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
				),
				ResourceName:      "cloudtemple_compute_iaas_opensource_virtual_machine.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"cloud_init",
					"template_id",
					"tools",
				},
			},
		},
	})
}

const testAccResourceIaasOpensourceVirtualMachine = `
data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_iaas_opensource_virtual_machine" "foo" {
  name        = "test-terraform-iaas-opensource-vm"
  template_id = "%s"
  cpu         = 2
  memory      = 2147483648
  power_state = "on"
  boot_firmware = "bios"
  auto_power_on = true

  tags = {
    "environment" = "test"
    "managed-by"  = "terraform"
  }

	backup_sla_policies = [
		data.cloudtemple_backup_iaas_opensource_policy.nobackup.id
	]

	lifecycle {
    ignore_changes = [
      memory,
    ]
  }
}
`

const testAccResourceIaasOpensourceVirtualMachineUpdate = `
data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_iaas_opensource_virtual_machine" "foo" {
  name        = "test-terraform-iaas-opensource-vm-updated"
  template_id = "%s"
  cpu         = 4
  memory      = 4294967296
  power_state = "on"
  boot_firmware = "bios"
  auto_power_on = true

  tags = {
    "environment" = "test"
    "managed-by"  = "terraform"
    "updated"     = "true"
  }

	backup_sla_policies = [
		data.cloudtemple_backup_iaas_opensource_policy.nobackup.id
	]

	lifecycle {
    ignore_changes = [
      memory,
    ]
  }
}
`
