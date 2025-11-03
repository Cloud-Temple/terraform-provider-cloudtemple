package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceObjectStorageRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceObjectStorageRole,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_object_storage_role.test", "name", "READ"),
				),
			},
		},
	})
}

const testAccDataSourceObjectStorageRole = `
data "cloudtemple_object_storage_role" "test" {
  name = "READ"
}
`
