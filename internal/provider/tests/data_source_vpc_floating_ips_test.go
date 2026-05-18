package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVPCFloatingIPs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVPCFloatingIPs,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_vpc_floating_ips.foo", "id", "floating_ips"),
				),
			},
		},
	})
}

const testAccDataSourceVPCFloatingIPs = `
data "cloudtemple_vpc_floating_ips" "foo" {}
`
