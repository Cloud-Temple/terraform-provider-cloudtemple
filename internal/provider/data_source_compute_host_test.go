package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceHost,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host.foo", "id", "8997db63-24d5-47f4-8cca-d5f5df199d1a"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host.foo", "name", "esx001-bob-ucs01-eqx6.cloud-temple.lan"),
				),
			},
			{
				Config:      testAccDataSourceHostMissing,
				ExpectError: regexp.MustCompile("failed to find host with id"),
			},
		},
	})
}

const testAccDataSourceHost = `
data "cloudtemple_compute_host" "foo" {
  id = "8997db63-24d5-47f4-8cca-d5f5df199d1a"
}
`

const testAccDataSourceHostMissing = `
data "cloudtemple_compute_host" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
