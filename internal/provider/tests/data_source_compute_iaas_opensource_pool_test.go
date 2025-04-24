package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSPoolId   = "COMPUTE_IAAS_OPENSOURCE_POOL_ID"
	OpenIaaSPoolName = "COMPUTE_IAAS_OPENSOURCE_POOL_NAME"
)

func TestAccDataSourceOpenIaaSPool(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSPool, os.Getenv(OpenIaaSPoolId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pool.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pool.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pool.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pool.foo", "machine_manager_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSPoolName, os.Getenv(OpenIaaSPoolName), os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pool.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pool.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pool.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pool.foo", "machine_manager_id"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSPoolMissing,
				ExpectError: regexp.MustCompile("failed to find pool with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSPool = `
data "cloudtemple_compute_iaas_opensource_pool" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSPoolName = `
data "cloudtemple_compute_iaas_opensource_pool" "foo" {
  name               = "%s"
  machine_manager_id = "%s"
}
`

const testAccDataSourceOpenIaaSPoolMissing = `
data "cloudtemple_compute_iaas_opensource_pool" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
