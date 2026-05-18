package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	StaticIPId = "STATIC_IP_ID"
)

func TestAccDataSourceVPCStaticIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVPCStaticIP, os.Getenv(StaticIPId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_vpc_static_ip.foo", "id", os.Getenv(StaticIPId)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_static_ip.foo", "ip_address"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_static_ip.foo", "mac_address"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_static_ip.foo", "source"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_static_ip.foo", "vpc_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_static_ip.foo", "private_network_id"),
				),
			},
			{
				Config:      testAccDataSourceVPCStaticIPMissing,
				ExpectError: regexp.MustCompile("failed to find static IP with id"),
			},
		},
	})
}

const testAccDataSourceVPCStaticIP = `
data "cloudtemple_vpc_static_ip" "foo" {
  id = "%s"
}
`

const testAccDataSourceVPCStaticIPMissing = `
data "cloudtemple_vpc_static_ip" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
