package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceObjectStorageStorageAccounts(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceObjectStorageStorageAccounts,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_object_storage_storage_accounts.foo", "storage_accounts.#"),
				),
			},
		},
	})
}

const testAccDataSourceObjectStorageStorageAccounts = `
data "cloudtemple_object_storage_storage_accounts" "foo" {}
`
