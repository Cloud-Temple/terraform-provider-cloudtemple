package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMRegions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMRegions,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_regions.all", "regions.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_regions.all", "regions.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_regions.all", "regions.0.name"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMRegions = `
data "cloudtemple_public_cloud_vm_regions" "all" {}
`
