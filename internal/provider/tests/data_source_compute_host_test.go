package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	HostId   = "COMPUTE_HOST_ID"
	HostName = "COMPUTE_HOST_NAME"
)

func TestAccDataSourceHost(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceHost, os.Getenv(HostId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host.foo", "id", os.Getenv(HostId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host.foo", "name", os.Getenv(HostName)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceHostName, os.Getenv(HostName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host.foo", "id", os.Getenv(HostId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host.foo", "name", os.Getenv(HostName)),
				),
			},
			{
				Config:      testAccDataSourceHostMissing,
				ExpectError: regexp.MustCompile("failed to find host with id"),
			},
		},
	})
}

const testAccDataSourceHost = `
data "cloudtemple_compute_host" "foo" {
  id = "%s"
}
`

const testAccDataSourceHostName = `
data "cloudtemple_compute_host" "foo" {
  name = "%s"
}
`

const testAccDataSourceHostMissing = `
data "cloudtemple_compute_host" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
