package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	NetworksQty = "COMPUTE_NETWORK_QTY"
)

func TestAccDataSourceNetworks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNetworks,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_networks.foo", "networks.#", os.Getenv(NetworksQty)),
				),
			},
		},
	})
}

const testAccDataSourceNetworks = `
data "cloudtemple_compute_networks" "foo" {}
`
