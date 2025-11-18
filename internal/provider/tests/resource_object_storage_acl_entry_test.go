package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceObjectStorageACLEntry_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceObjectStorageACLEntryBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_object_storage_acl_entry.test", "bucket", "test-bucket"),
					resource.TestCheckResourceAttr("cloudtemple_object_storage_acl_entry.test", "role", "READ"),
					resource.TestCheckResourceAttr("cloudtemple_object_storage_acl_entry.test", "storage_account", "test-storage-account"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_acl_entry.test", "id"),
				),
			},
		},
	})
}

const testAccResourceObjectStorageACLEntryBasic = `
resource "cloudtemple_object_storage_acl_entry" "test" {
  bucket          = "test-bucket"
  role            = "READ"
  storage_account = "test-storage-account"
}
`
