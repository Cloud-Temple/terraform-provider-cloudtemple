package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualMachine,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_machine.foo", "id", "de2b8b80-8b90-414a-bc33-e12f61a4c05c"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_machine.foo", "name", "virtual_machine_67_bob-clone"),
				),
			},
			{
				Config: testAccDataSourceVirtualMachineName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_machine.foo", "id", "de2b8b80-8b90-414a-bc33-e12f61a4c05c"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_machine.foo", "name", "virtual_machine_67_bob-clone"),
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
  id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}
`

const testAccDataSourceVirtualMachineName = `
data "cloudtemple_compute_virtual_machine" "foo" {
  name = "virtual_machine_67_bob-clone"
}
`

const testAccDataSourceVirtualMachineMissing = `
data "cloudtemple_compute_virtual_machine" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
