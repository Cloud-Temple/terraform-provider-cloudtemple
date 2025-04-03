package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	HostsQty = "COMPUTE_HOST_QTY"
)

func TestAccDataSourceHosts(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceHosts,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_hosts.foo", "hosts.#", os.Getenv(HostsQty)),
				),
			},
		},
	})
}

const testAccDataSourceHosts = `
data "cloudtemple_compute_hosts" "foo" {}
`
