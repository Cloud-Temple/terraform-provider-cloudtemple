package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualDisks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualDisks,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.#", "1"),
				),
			},
			{
				Config: testAccDataSourceVirtualDisksMissing,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualDisks = `
data "cloudtemple_compute_virtual_disks" "foo" {
  virtual_machine_id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}
`

const testAccDataSourceVirtualDisksMissing = `
data "cloudtemple_compute_virtual_disks" "foo" {
  virtual_machine_id = "12345678-1234-5678-1234-567812345678"
}
`
