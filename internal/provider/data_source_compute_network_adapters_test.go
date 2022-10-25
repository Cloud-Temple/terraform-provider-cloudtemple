package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceNetworkAdapters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNetworkAdapters,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapters.foo", "network_adapters.#", "1"),
				),
			},
			{
				Config: testAccDataSourceNetworkAdaptersMissing,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapters.foo", "network_adapters.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceNetworkAdapters = `
data "cloudtemple_compute_network_adapters" "foo" {
  virtual_machine_id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}
`

const testAccDataSourceNetworkAdaptersMissing = `
data "cloudtemple_compute_network_adapters" "foo" {
  virtual_machine_id = "12345678-1234-5678-1234-567812345678"
}
`
