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
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_machine.foo", "id", "dba8aea7-7718-4ffb-8932-9acf4c8cc629"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_machine.foo", "name", "tf-do-not-delete"),
				),
			},
			{
				Config: testAccDataSourceVirtualMachineName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_machine.foo", "id", "dba8aea7-7718-4ffb-8932-9acf4c8cc629"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_machine.foo", "name", "tf-do-not-delete"),
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
  id = "dba8aea7-7718-4ffb-8932-9acf4c8cc629"
}
`

const testAccDataSourceVirtualMachineName = `
data "cloudtemple_compute_virtual_machine" "foo" {
  name = "tf-do-not-delete"
}
`

const testAccDataSourceVirtualMachineMissing = `
data "cloudtemple_compute_virtual_machine" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
