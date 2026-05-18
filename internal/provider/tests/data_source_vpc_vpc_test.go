package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VPCId   = "VPC_ID"
	VPCName = "VPC_NAME"
)

func TestAccDataSourceVPCVPC(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVPCVPC, os.Getenv(VPCId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_vpc_vpc.foo", "id", os.Getenv(VPCId)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_vpc.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_vpc.foo", "private_network_count"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_vpc.foo", "static_ip_count"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_vpc_vpc.foo", "floating_ip_count"),
				),
			},
			{
				Config:      testAccDataSourceVPCVPCMissing,
				ExpectError: regexp.MustCompile("failed to find VPC with id"),
			},
		},
	})
}

const testAccDataSourceVPCVPC = `
data "cloudtemple_vpc_vpc" "foo" {
  id = "%s"
}
`

const testAccDataSourceVPCVPCMissing = `
data "cloudtemple_vpc_vpc" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
