package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OperatingSystemMoRef = "COMPUTE_OPERATING_SYSTEM_MOREF"
)

func TestAccDataSourceGuestOperatingSystem(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceGuestOperatingSystem, os.Getenv(OperatingSystemMoRef), os.Getenv(HostClusterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_guest_operating_system.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_guest_operating_system.foo", "family"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_guest_operating_system.foo", "full_name"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourceGuestOperatingSystemMissing),
				ExpectError: regexp.MustCompile("Access denied: 403 Forbidden"),
			},
		},
	})
}

const testAccDataSourceGuestOperatingSystem = `
data "cloudtemple_compute_guest_operating_system" "foo" {
  moref              = "%s"
  host_cluster_id = "%s"
}
`

const testAccDataSourceGuestOperatingSystemMissing = `
data "cloudtemple_compute_guest_operating_system" "foo" {
	moref              = "invalid-moref"
  host_cluster_id = "f7336dc8-7a91-461f-933d-3642aa415446"
}
`
