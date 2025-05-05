package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ResourcePoolId   = "COMPUTE_RESOURCE_POOL_ID"
	ResourcePoolName = "COMPUTE_RESOURCE_POOL_NAME"
)

func TestAccDataSourceResourcePool(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceResourcePool, os.Getenv(ResourcePoolId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "parent.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "parent.0.type"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceResourcePoolName, os.Getenv(ResourcePoolName), os.Getenv(MachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "parent.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pool.foo", "parent.0.type"),
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
	machine_manager_id = "%s"
}
`

const testAccDataSourceResourcePoolMissing = `
data "cloudtemple_compute_resource_pool" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
