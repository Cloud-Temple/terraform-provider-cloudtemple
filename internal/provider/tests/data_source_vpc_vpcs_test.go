package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVPCVPCs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVPCVPCs,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_vpc_vpcs.foo", "id", "vpcs"),
				),
			},
		},
	})
}

const testAccDataSourceVPCVPCs = `
data "cloudtemple_vpc_vpcs" "foo" {}
`
