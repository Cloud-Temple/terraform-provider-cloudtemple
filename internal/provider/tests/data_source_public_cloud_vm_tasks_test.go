package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMTasks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePublicCloudVMTasks,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_tasks.recent", "tasks.#"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMTasks = `
data "cloudtemple_public_cloud_vm_tasks" "recent" {
  limit = 20
}
`
