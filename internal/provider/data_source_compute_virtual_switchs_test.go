package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VirtualSwitchQty = "COMPUTE_VIRTUAL_SWITCH_QTY"
)

func TestAccDataSourceVirtualSwitchs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualSwitchs,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_switchs.foo", "virtual_switchs.#", os.Getenv(VirtualSwitchQty)),
				),
			},
		},
	})
}

const testAccDataSourceVirtualSwitchs = `
data "cloudtemple_compute_virtual_switchs" "foo" {}
`
