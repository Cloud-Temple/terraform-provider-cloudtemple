package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VirtualMachineId   = "COMPUTE_VIRTUAL_MACHINE_ID"
	VirtualMachineName = "COMPUTE_VIRTUAL_MACHINE_NAME"
)

func TestAccDataSourceVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualMachine, os.Getenv(VirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "power_state"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualMachineName, os.Getenv(VirtualMachineName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machine.foo", "power_state"),
				),
			},
			{
				Config:      testAccDataSourceVirtualMachineMissing,
				ExpectError: regexp.MustCompile("failed to find virtual machine with id"),
			},
		},
	})
}

const testAccDataSourceVirtualMachine = `
data "cloudtemple_compute_virtual_machine" "foo" {
  id = "%s"
}
`

const testAccDataSourceVirtualMachineName = `
data "cloudtemple_compute_virtual_machine" "foo" {
  name = "%s"
}
`

const testAccDataSourceVirtualMachineMissing = `
data "cloudtemple_compute_virtual_machine" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
