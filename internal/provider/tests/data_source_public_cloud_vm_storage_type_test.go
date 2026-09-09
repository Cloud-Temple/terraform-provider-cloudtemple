package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMStorageTypeName = "PUBLIC_CLOUD_VM_STORAGE_TYPE_NAME"
)

func TestAccDataSourcePublicCloudVMStorageType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMStorageTypeName, os.Getenv(PublicCloudVMStorageTypeName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_type.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_storage_type.foo", "name", os.Getenv(PublicCloudVMStorageTypeName)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_type.foo", "max_size_gib"),
					// The API now returns a priced SKU on each storage type (#507).
					// Assert presence, not the live price value (which is volatile).
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_type.foo", "sku.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_type.foo", "sku.0.price"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_type.foo", "sku.0.unit"),
				),
			},
			{
				Config:      testAccDataSourcePublicCloudVMStorageTypeMissing,
				ExpectError: regexp.MustCompile("failed to find storage type with id"),
			},
			{
				// Name resolved within a real availability-zone/instance-family
				// pair, the family taken from the zone's compatible_families
				// (valid pair by construction). Pins the paired filters and the
				// provider-side pair validation end-to-end — NOT server-side
				// narrowing, which the client unit test pins at the wire level.
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMStorageTypeFiltered, os.Getenv(PublicCloudVMStorageTypeName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_storage_type.narrowed", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_storage_type.narrowed", "name", os.Getenv(PublicCloudVMStorageTypeName)),
				),
			},
			{
				// A lone filter must be refused at plan time (RequiredWith),
				// before any API call.
				Config:      fmt.Sprintf(testAccDataSourcePublicCloudVMStorageTypeLoneFilter, os.Getenv(PublicCloudVMStorageTypeName)),
				ExpectError: regexp.MustCompile(`all of\s+.availability_zone_id,instance_family_id.\s+must be specified`),
				PlanOnly:    true,
			},
		},
	})
}

const testAccDataSourcePublicCloudVMStorageTypeName = `
data "cloudtemple_public_cloud_vm_storage_type" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMStorageTypeMissing = `
data "cloudtemple_public_cloud_vm_storage_type" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`

const testAccDataSourcePublicCloudVMStorageTypeFiltered = `
data "cloudtemple_public_cloud_vm_availability_zones" "all" {}

data "cloudtemple_public_cloud_vm_storage_type" "narrowed" {
  name                 = "%s"
  availability_zone_id = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].id
  instance_family_id   = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].compatible_families[0].id
}
`

const testAccDataSourcePublicCloudVMStorageTypeLoneFilter = `
data "cloudtemple_public_cloud_vm_instance_families" "all" {}

data "cloudtemple_public_cloud_vm_storage_type" "lone" {
  name               = "%s"
  instance_family_id = data.cloudtemple_public_cloud_vm_instance_families.all.instance_families[0].id
}
`
