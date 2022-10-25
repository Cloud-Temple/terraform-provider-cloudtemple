package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceNetworkAdapter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNetworkAdapter,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "id", "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "name", "Network adapter 1"),
				),
			},
		},
	})
}

const testAccDataSourceNetworkAdapter = `
data "cloudtemple_compute_network_adapter" "foo" {
  id = "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86"
}
`
