package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMTemplates(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMTemplates,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_templates.all", "templates.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_templates.all", "templates.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_templates.all", "templates.0.os_name"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMTemplates = `
data "cloudtemple_public_cloud_vm_templates" "all" {}
`
