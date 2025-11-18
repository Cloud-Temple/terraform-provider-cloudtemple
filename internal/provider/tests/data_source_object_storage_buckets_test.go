package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceObjectStorageBuckets(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceObjectStorageBuckets,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_buckets.foo", "buckets.#"),
				),
			},
		},
	})
}

const testAccDataSourceObjectStorageBuckets = `
data "cloudtemple_object_storage_buckets" "foo" {}
`
