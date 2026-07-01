package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourcePublicCloudVMInstanceSnapshots(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMSnapshots, os.Getenv(PublicCloudVMInstanceId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The count attribute is always set (0 or more); listing must not error.
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_snapshots.all", "snapshots.#"),
				),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMSnapshots = `
data "cloudtemple_public_cloud_vm_snapshots" "all" {
  virtual_machine_id = "%s"
}
`
