package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceResourcePool(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceResourcePool,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_resource_pool.foo", "id", "d21f84fd-5063-4383-b2b0-65b9f25eac27"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_resource_pool.foo", "name", "Resources"),
				),
			},
		},
	})
}

const testAccDataSourceResourcePool = `
data "cloudtemple_compute_resource_pool" "foo" {
  id = "d21f84fd-5063-4383-b2b0-65b9f25eac27"
}
`
