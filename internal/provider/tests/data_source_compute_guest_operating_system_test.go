package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OperatingSystemMoRef = "COMPUTE_OPERATION_SYSTEM_MOREF"
)

func TestAccDataSourceGuestOperatingSystem(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceGuestOperatingSystem, os.Getenv(OperatingSystemMoRef), os.Getenv(MachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_guest_operating_system.foo", "id", os.Getenv(OperatingSystemMoRef)),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourceGuestOperatingSystemMissing, os.Getenv(MachineManagerId)),
				ExpectError: regexp.MustCompile("Forbidden."),
			},
		},
	})
}

const testAccDataSourceGuestOperatingSystem = `
data "cloudtemple_compute_guest_operating_system" "foo" {
  moref              = "%s"
  machine_manager_id = "%s"
}
`

const testAccDataSourceGuestOperatingSystemMissing = `
data "cloudtemple_compute_guest_operating_system" "foo" {
  moref              = "%s"
  machine_manager_id = "12345678-1234-5678-1234-567812345678"
}
`
