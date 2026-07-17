package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMFlavors(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMFlavors,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_flavors.all", "flavors.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_flavors.all", "flavors.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_flavors.all", "flavors.0.vcpu"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMFlavors = `
data "cloudtemple_public_cloud_vm_flavors" "all" {}
`
