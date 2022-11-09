package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceGuestOperatingSystems(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceGuestOperatingSystems,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_guest_operating_systems.foo", "guest_operating_systems.#"),
				),
			},
			{
				Config: testAccDataSourceGuestOperatingSystemsMissing,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_guest_operating_systems.foo", "guest_operating_systems.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceGuestOperatingSystems = `
data "cloudtemple_compute_guest_operating_systems" "foo" {
  machine_manager_id = "9dba240e-a605-4103-bac7-5336d3ffd124"
}
`

const testAccDataSourceGuestOperatingSystemsMissing = `
data "cloudtemple_compute_guest_operating_systems" "foo" {
  machine_manager_id = "12345678-1234-5678-1234-567812345678"
}
`
