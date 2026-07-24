package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMImages(t *testing.T) {
	skipIfNoPublicCloudVMImageEnv(t)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMImages,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_images.all", "images.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_images.all", "images.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_images.all", "images.0.os_name"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMImages = `
data "cloudtemple_public_cloud_vm_images" "all" {}
`
