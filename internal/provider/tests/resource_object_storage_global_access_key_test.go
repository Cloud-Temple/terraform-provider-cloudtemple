package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testAccResourceObjectStorageGlobalAccessKey = `
resource "cloudtemple_object_storage_global_access_key" "test" {
}
`

func TestAccResourceObjectStorageGlobalAccessKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceObjectStorageGlobalAccessKey,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_object_storage_global_access_key.test", "id", "global_access_key"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_global_access_key.test", "access_key_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_object_storage_global_access_key.test", "access_secret_key"),
				),
			},
		},
	})
}

func init() {
	resource.AddTestSweepers("cloudtemple_object_storage_global_access_key", &resource.Sweeper{
		Name: "cloudtemple_object_storage_global_access_key",
		F: func(string) error {
			// Note: The global access key cannot be deleted via API
			// This sweeper is a no-op as the resource is only removed from state on destroy
			return nil
		},
	})
}
