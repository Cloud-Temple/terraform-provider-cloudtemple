package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualDisk,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "id", "d370b8cd-83eb-4315-a5d9-42157e2e4bb4"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "name", "Hard disk 1"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualDisk = `
data "cloudtemple_compute_virtual_disk" "foo" {
  id = "d370b8cd-83eb-4315-a5d9-42157e2e4bb4"
}
`
