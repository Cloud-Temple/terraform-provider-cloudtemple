package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	FloatingIPId = "FLOATING_IP_ID"
)

func TestAccDataSourceVPCFloatingIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVPCFloatingIP, os.Getenv(FloatingIPId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_vpc_floating_ip.foo", "id", os.Getenv(FloatingIPId)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_floating_ip.foo", "ip_address"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_floating_ip.foo", "description"),
				),
			},
			{
				Config:      testAccDataSourceVPCFloatingIPMissing,
				ExpectError: regexp.MustCompile("failed to find floating IP with id"),
			},
		},
	})
}

const testAccDataSourceVPCFloatingIP = `
data "cloudtemple_vpc_floating_ip" "foo" {
  id = "%s"
}
`

const testAccDataSourceVPCFloatingIPMissing = `
data "cloudtemple_vpc_floating_ip" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
