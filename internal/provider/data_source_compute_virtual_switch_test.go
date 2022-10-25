package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualSwitch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualSwitch,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_switch.foo", "id", "6e7b457c-bdb1-4272-8abf-5fd6e9adb8a4"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_switch.foo", "name", "dvs002-ucs01_FLO-DC-EQX6"),
				),
			},
			{
				Config:      testAccDataSourceVirtualSwitchMissing,
				ExpectError: regexp.MustCompile("failed to find virtual switch with id"),
			},
		},
	})
}

const testAccDataSourceVirtualSwitch = `
data "cloudtemple_compute_virtual_switch" "foo" {
  id = "6e7b457c-bdb1-4272-8abf-5fd6e9adb8a4"
}
`

const testAccDataSourceVirtualSwitchMissing = `
data "cloudtemple_compute_virtual_switch" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
