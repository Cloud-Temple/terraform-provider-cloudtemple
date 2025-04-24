package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSAvailabilityZoneId   = "COMPUTE_IAAS_OPENSOURCE_AVAILABILITY_ZONE_ID"
	OpenIaaSAvailabilityZoneName = "COMPUTE_IAAS_OPENSOURCE_AVAILABILITY_ZONE_NAME"
)

func TestAccDataSourceOpenIaaSAvailabilityZone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSAvailabilityZone, os.Getenv(OpenIaaSAvailabilityZoneId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zone.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zone.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zone.foo", "os_name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zone.foo", "os_version"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSAvailabilityZoneName, os.Getenv(OpenIaaSAvailabilityZoneName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zone.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zone.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zone.foo", "os_name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_availability_zone.foo", "os_version"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSAvailabilityZoneMissing,
				ExpectError: regexp.MustCompile("failed to find availability zone with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSAvailabilityZone = `
data "cloudtemple_compute_iaas_opensource_availability_zone" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSAvailabilityZoneName = `
data "cloudtemple_compute_iaas_opensource_availability_zone" "foo" {
  name = "%s"
}
`

const testAccDataSourceOpenIaaSAvailabilityZoneMissing = `
data "cloudtemple_compute_iaas_opensource_availability_zone" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
