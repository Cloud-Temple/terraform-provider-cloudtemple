package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceObjectStorageRoles(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceObjectStorageRoles,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_roles.test", "roles.#"),
				),
			},
		},
	})
}

const testAccDataSourceObjectStorageRoles = `
data "cloudtemple_object_storage_roles" "test" {}
`
