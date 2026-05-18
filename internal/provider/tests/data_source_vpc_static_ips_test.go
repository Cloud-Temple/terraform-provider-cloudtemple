package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVPCStaticIPs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVPCStaticIPs, os.Getenv(PrivateNetworkId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_vpc_static_ips.foo", "id", "static_ips"),
				),
			},
		},
	})
}

const testAccDataSourceVPCStaticIPs = `
data "cloudtemple_vpc_static_ips" "foo" {
  private_network_id = "%s"
}
`
