package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMInstanceFamilies(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMInstanceFamilies,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance_families.all", "instance_families.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance_families.all", "instance_families.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instance_families.all", "instance_families.0.vcpu_max"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMInstanceFamilies = `
data "cloudtemple_public_cloud_vm_instance_families" "all" {}
`
