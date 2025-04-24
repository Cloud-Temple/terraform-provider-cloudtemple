package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceIaasOpensourceVirtualDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualDisk,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_STORAGE_REPOSITORY_ID"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_disk.disk", "name", "test-terraform-iaas-opensource-disk"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_disk.disk", "size", "10737418240"), // 10GB
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_disk.disk", "mode", "RW"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_disk.disk", "bootable", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_disk.disk", "internal_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_disk.disk", "usage"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_disk.disk", "is_snapshot"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualDisk,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_STORAGE_REPOSITORY_ID"),
				),
				ResourceName:      "cloudtemple_compute_iaas_opensource_virtual_disk.disk",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"mode",
					"virtual_machine_id",
					"bootable",
				},
			},
		},
	})
}

const testAccResourceIaasOpensourceVirtualDisk = `
data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vm" {
  name        = "test-terraform-iaas-opensource-vm-for-disk"
  template_id = "%s"
  cpu         = 2
  memory      = 2147483648
  power_state = "on"

	backup_sla_policies = [
		data.cloudtemple_backup_iaas_opensource_policy.nobackup.id
	]

	lifecycle {
		ignore_changes = [
			"memory",
		]
	}
}

resource "cloudtemple_compute_iaas_opensource_virtual_disk" "disk" {
  name                  = "test-terraform-iaas-opensource-disk"
  size                  = 10737418240 # 10GB
  mode                  = "RW"
  storage_repository_id = "%s"
  virtual_machine_id    = cloudtemple_compute_iaas_opensource_virtual_machine.vm.id
  bootable              = false
}
`
