package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMNetworks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMNetworks,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_networks.all", "networks.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_networks.all", "networks.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_networks.all", "networks.0.name"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMNetworks = `
data "cloudtemple_public_cloud_vm_networks" "all" {}
`
