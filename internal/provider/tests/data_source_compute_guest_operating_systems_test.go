package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceGuestOperatingSystems(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceGuestOperatingSystems, os.Getenv(MachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_guest_operating_systems.foo", "guest_operating_systems.#"),
				),
			},
			{
				Config: testAccDataSourceGuestOperatingSystemsMissing,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_guest_operating_systems.foo", "guest_operating_systems.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceGuestOperatingSystems = `
data "cloudtemple_compute_guest_operating_systems" "foo" {
  machine_manager_id = "%s"
}
`

const testAccDataSourceGuestOperatingSystemsMissing = `
data "cloudtemple_compute_guest_operating_systems" "foo" {
  machine_manager_id = "12345678-1234-5678-1234-567812345678"
}
`
