package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualDatacenters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualDatacenters,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_datacenters.foo", "virtual_datacenters.#", "2"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualDatacenters = `
data "cloudtemple_compute_virtual_datacenters" "foo" {}
`
