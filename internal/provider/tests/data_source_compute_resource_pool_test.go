package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ResourcePoolId   = "COMPTE_RESOURCE_POOL_ID"
	ResourcePoolName = "COMPTE_RESOURCE_POOL_NAME"
)

func TestAccDataSourceResourcePool(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceResourcePool, os.Getenv(ResourcePoolId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_resource_pool.foo", "id", os.Getenv(ResourcePoolId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_resource_pool.foo", "name", os.Getenv(ResourcePoolName)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceResourcePoolName, os.Getenv(ResourcePoolName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_resource_pool.foo", "id", os.Getenv(ResourcePoolId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_resource_pool.foo", "name", os.Getenv(ResourcePoolName)),
				),
			},
			{
				Config:      testAccDataSourceResourcePoolMissing,
				ExpectError: regexp.MustCompile("failed to find resource pool with id"),
			},
		},
	})
}

const testAccDataSourceResourcePool = `
data "cloudtemple_compute_resource_pool" "foo" {
  id = "%s"
}
`

const testAccDataSourceResourcePoolName = `
data "cloudtemple_compute_resource_pool" "foo" {
  name = "%s"
}
`

const testAccDataSourceResourcePoolMissing = `
data "cloudtemple_compute_resource_pool" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
