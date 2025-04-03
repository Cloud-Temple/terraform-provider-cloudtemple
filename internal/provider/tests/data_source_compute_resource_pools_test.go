package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ResourcePoolQty = "COMPTE_RESOURCE_POOL_QTY"
)

func TestAccDataSourceResourcePools(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceResourcePools,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_resource_pools.foo", "resource_pools.#", os.Getenv(ResourcePoolQty)),
				),
			},
		},
	})
}

const testAccDataSourceResourcePools = `
data "cloudtemple_compute_resource_pools" "foo" {}
`
