package provider

import (
	"regexp"
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
			{
				// Filtered by a real availability-zone/instance-family pair
				// (the family is taken from the zone's compatible_families, so
				// the pair is valid by construction). Pins that the paired
				// filters and the provider-side pair validation are accepted
				// end-to-end. It does NOT prove server-side narrowing: the
				// wire spelling is pinned by the client unit test.
				Config: testAccDataSourcePublicCloudVMStorageTypesFiltered,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_types.filtered", "storage_types.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_types.filtered", "storage_types.0.id"),
				),
			},
			{
				// A lone filter must be refused at plan time (RequiredWith),
				// before any API call.
				Config:      testAccDataSourcePublicCloudVMStorageTypesLoneFilter,
				ExpectError: regexp.MustCompile(`all of\s+.availability_zone_id,instance_family_id.\s+must be specified`),
				PlanOnly:    true,
			},
		},
	})
}

const testAccDataSourcePublicCloudVMStorageTypes = `
data "cloudtemple_public_cloud_vm_storage_types" "all" {}
`

const testAccDataSourcePublicCloudVMStorageTypesFiltered = `
data "cloudtemple_public_cloud_vm_availability_zones" "all" {}

data "cloudtemple_public_cloud_vm_storage_types" "filtered" {
  availability_zone_id = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].id
  instance_family_id   = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].compatible_families[0].id
}
`

const testAccDataSourcePublicCloudVMStorageTypesLoneFilter = `
data "cloudtemple_public_cloud_vm_availability_zones" "all" {}

data "cloudtemple_public_cloud_vm_storage_types" "lone" {
  availability_zone_id = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].id
}
`
