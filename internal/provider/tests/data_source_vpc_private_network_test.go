package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PrivateNetworkId = "PRIVATE_NETWORK_ID"
)

func TestAccDataSourceVPCPrivateNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVPCPrivateNetwork, os.Getenv(PrivateNetworkId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_vpc_private_network.foo", "id", os.Getenv(PrivateNetworkId)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_private_network.foo", "ip_address"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_private_network.foo", "vlan_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_private_network.foo", "static_ip_count"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_private_network.foo", "vpc_id"),
				),
			},
			{
				Config:      testAccDataSourceVPCPrivateNetworkMissing,
				ExpectError: regexp.MustCompile("failed to find private network with id"),
			},
		},
	})
}

const testAccDataSourceVPCPrivateNetwork = `
data "cloudtemple_vpc_private_network" "foo" {
  id = "%s"
}
`

const testAccDataSourceVPCPrivateNetworkMissing = `
data "cloudtemple_vpc_private_network" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
