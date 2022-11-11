package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNetwork,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network.foo", "id", "5e029210-b433-4c45-93be-092cef684edc"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network.foo", "name", "VLAN_201"),
				),
			},
			{
				Config: testAccDataSourceNetworkName,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network.foo", "id", "5e029210-b433-4c45-93be-092cef684edc"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network.foo", "name", "VLAN_201"),
				),
			},
			{
				Config:      testAccDataSourceNetworkMissing,
				ExpectError: regexp.MustCompile("failed to find network with id"),
			},
		},
	})
}

const testAccDataSourceNetwork = `
data "cloudtemple_compute_network" "foo" {
  id = "5e029210-b433-4c45-93be-092cef684edc"
}
`

const testAccDataSourceNetworkName = `
data "cloudtemple_compute_network" "foo" {
  name = "VLAN_201"
}
`

const testAccDataSourceNetworkMissing = `
data "cloudtemple_compute_network" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
