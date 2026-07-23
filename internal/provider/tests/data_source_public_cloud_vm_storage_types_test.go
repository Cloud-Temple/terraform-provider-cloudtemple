package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMStorageTypes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMStorageTypes,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_types.all", "storage_types.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_types.all", "storage_types.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_types.all", "storage_types.0.max_size_gb"),
					// The API now returns a priced SKU on each storage type (#507).
					// Assert presence, not the live price value (which is volatile).
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_types.all", "storage_types.0.sku.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_types.all", "storage_types.0.sku.0.price"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMStorageTypes = `
data "cloudtemple_public_cloud_vm_storage_types" "all" {}
`
