package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSVirtualMachineName = "COMPUTE_IAAS_OPENSOURCE_VIRTUAL_MACHINE_NAME"
)

func TestAccDataSourceOpenIaaSVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSVirtualMachine, os.Getenv(OpenIaaSVirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "power_state"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSVirtualMachineName, os.Getenv(OpenIaaSVirtualMachineName), os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_machine.foo", "power_state"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSVirtualMachineMissing,
				ExpectError: regexp.MustCompile("failed to find virtual machine with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSVirtualMachine = `
data "cloudtemple_compute_iaas_opensource_virtual_machine" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSVirtualMachineName = `
data "cloudtemple_compute_iaas_opensource_virtual_machine" "foo" {
  name               = "%s"
  machine_manager_id = "%s"
}
`

const testAccDataSourceOpenIaaSVirtualMachineMissing = `
data "cloudtemple_compute_iaas_opensource_virtual_machine" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
