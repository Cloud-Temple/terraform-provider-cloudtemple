package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualSwitchs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualSwitchs,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_switchs.foo", "virtual_switchs.#", "2"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualSwitchs = `
data "cloudtemple_compute_virtual_switchs" "foo" {}
`
