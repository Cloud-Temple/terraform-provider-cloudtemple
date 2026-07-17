package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMInstanceDisks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMDisks, os.Getenv(PublicCloudVMInstanceId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_disks.all", "disks.#"),
					// Every VM has at least a system disk.
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_disks.all", "disks.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_disks.all", "disks.0.size_gb"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMDisks = `
data "cloudtemple_public_cloud_vm_disks" "all" {
  virtual_machine_id = "%s"
}
`
