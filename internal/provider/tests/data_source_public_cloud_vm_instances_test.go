package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMInstances(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMInstances,
				Check: resource.ComposeAggregateTestCheckFunc(
					// The count attribute is always set (0 or more); the listing must
					// not error even on an empty tenant.
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_instances.all", "instances.#"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMInstances = `
data "cloudtemple_public_cloud_vm_instances" "all" {}
`
