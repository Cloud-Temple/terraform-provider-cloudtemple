package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	NetworkId   = "COMPUTE_NETWORK_ID"
	NetworkName = "COMPUTE_NETWORK_NAME"
)

func TestAccDataSourceNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceNetwork, os.Getenv(NetworkId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "virtual_machines_number"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "host_number"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceNetworkName, os.Getenv(NetworkName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "virtual_machines_number"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network.foo", "host_number"),
				),
			},
			{
				Config:      testAccDataSourceNetworkMissing,
				ExpectError: regexp.MustCompile("failed to find virtual network with id"),
			},
		},
	})
}

const testAccDataSourceNetwork = `
data "cloudtemple_compute_network" "foo" {
  id = "%s"
}
`

const testAccDataSourceNetworkName = `
data "cloudtemple_compute_network" "foo" {
  name = "%s"
}
`

const testAccDataSourceNetworkMissing = `
data "cloudtemple_compute_network" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
