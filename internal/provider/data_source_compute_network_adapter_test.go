package provider

import (
	"regexp"
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
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "virtual_machine_id", "de2b8b80-8b90-414a-bc33-e12f61a4c05c"),
				),
			},
			{
				Config: testAccDataSourceNetworkAdapterName,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "id", "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "name", "Network adapter 1"),
				),
			},
			{
				Config:      testAccDataSourceNetworkAdapterMissing,
				ExpectError: regexp.MustCompile("failed to find network adapter with id"),
			},
		},
	})
}

const testAccDataSourceNetworkAdapter = `
data "cloudtemple_compute_network_adapter" "foo" {
  id = "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86"
}
`

const testAccDataSourceNetworkAdapterName = `
data "cloudtemple_compute_network_adapter" "foo" {
  name               = "Network adapter 1"
  virtual_machine_id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}
`

const testAccDataSourceNetworkAdapterMissing = `
data "cloudtemple_compute_network_adapter" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
