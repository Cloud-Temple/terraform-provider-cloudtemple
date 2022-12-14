package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceGuestOperatingSystem(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceGuestOperatingSystem,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_guest_operating_system.foo", "id", "amazonlinux2_64Guest"),
				),
			},
			{
				Config:      testAccDataSourceGuestOperatingSystemMissing,
				ExpectError: regexp.MustCompile("failed to find guest operating system"),
			},
		},
	})
}

const testAccDataSourceGuestOperatingSystem = `
data "cloudtemple_compute_guest_operating_system" "foo" {
  moref              = "amazonlinux2_64Guest"
  machine_manager_id = "9dba240e-a605-4103-bac7-5336d3ffd124"
}
`

const testAccDataSourceGuestOperatingSystemMissing = `
data "cloudtemple_compute_guest_operating_system" "foo" {
  moref              = "amazonlinux2_64Guest"
  machine_manager_id = "12345678-1234-5678-1234-567812345678"
}
`
