package provider

import (
	"regexp"
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
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "virtual_machine_id", "de2b8b80-8b90-414a-bc33-e12f61a4c05c"),
				),
			}, {
				Config: testAccDataSourceVirtualDiskName,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "id", "d370b8cd-83eb-4315-a5d9-42157e2e4bb4"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "name", "Hard disk 1"),
				),
			},
			{
				Config:      testAccDataSourceVirtualDiskMissing,
				ExpectError: regexp.MustCompile("failed to find virtual disk with id"),
			},
		},
	})
}

const testAccDataSourceVirtualDisk = `
data "cloudtemple_compute_virtual_disk" "foo" {
  id = "d370b8cd-83eb-4315-a5d9-42157e2e4bb4"
}
`

const testAccDataSourceVirtualDiskName = `
data "cloudtemple_compute_virtual_disk" "foo" {
  name               = "Hard disk 1"
  virtual_machine_id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}
`

const testAccDataSourceVirtualDiskMissing = `
data "cloudtemple_compute_virtual_disk" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
