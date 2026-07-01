package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMRegionId   = "PUBLIC_CLOUD_VM_REGION_ID"
	PublicCloudVMRegionName = "PUBLIC_CLOUD_VM_REGION_NAME"
)

func TestAccDataSourcePublicCloudVMRegion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMRegion, os.Getenv(PublicCloudVMRegionId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_region.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_region.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_region.foo", "country_code"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMRegionName, os.Getenv(PublicCloudVMRegionName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_region.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_region.foo", "name", os.Getenv(PublicCloudVMRegionName)),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourcePublicCloudVMRegionConflict, os.Getenv(PublicCloudVMRegionId), os.Getenv(PublicCloudVMRegionName)),
				ExpectError: regexp.MustCompile(`"id": conflicts with name`),
			},
			{
				Config:      testAccDataSourcePublicCloudVMRegionMissing,
				ExpectError: regexp.MustCompile("failed to find region with id"),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMRegion = `
data "cloudtemple_public_cloud_vm_region" "foo" {
  id = "%s"
}
`

const testAccDataSourcePublicCloudVMRegionName = `
data "cloudtemple_public_cloud_vm_region" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMRegionConflict = `
data "cloudtemple_public_cloud_vm_region" "foo" {
  id   = "%s"
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMRegionMissing = `
data "cloudtemple_public_cloud_vm_region" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
