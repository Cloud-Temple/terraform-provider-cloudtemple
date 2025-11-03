package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceObjectStorageStorageAccount_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceObjectStorageStorageAccountBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_object_storage_storage_account.test", "name", "terraform-test-storage-account"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_storage_account.test", "id"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_storage_account.test", "access_key_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_storage_account.test", "access_secret_key"),
				),
			},
		},
	})
}

const testAccResourceObjectStorageStorageAccountBasic = `
resource "cloudtemple_object_storage_storage_account" "test" {
  name = "terraform-test-storage-account"
}
`
