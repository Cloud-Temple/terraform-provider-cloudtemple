package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualDatacenter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualDatacenter,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_datacenter.foo", "id", "ac33c033-693b-4fc5-9196-26df77291dbb"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_datacenter.foo", "name", "DC-TH3"),
				),
			},
			{
				Config:      testAccDataSourceVirtualDatacenterMissing,
				ExpectError: regexp.MustCompile("failed to find virtual datacenter with id"),
			},
		},
	})
}

const testAccDataSourceVirtualDatacenter = `
data "cloudtemple_compute_virtual_datacenter" "foo" {
  id = "ac33c033-693b-4fc5-9196-26df77291dbb"
}
`

const testAccDataSourceVirtualDatacenterMissing = `
data "cloudtemple_compute_virtual_datacenter" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
