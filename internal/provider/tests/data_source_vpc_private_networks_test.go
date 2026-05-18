package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVPCPrivateNetworks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVPCPrivateNetworks,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_vpc_private_networks.foo", "id", "private_networks"),
				),
			},
		},
	})
}

const testAccDataSourceVPCPrivateNetworks = `
data "cloudtemple_vpc_private_networks" "foo" {}
`
