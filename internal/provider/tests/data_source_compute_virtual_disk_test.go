package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VirtualDiskId   = "COMPUTE_VIRTUAL_DISK_ID"
	VirtualDiskName = "COMPUTE_VIRTUAL_DISK_NAME"
)

func TestAccDataSourceVirtualDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualDisk, os.Getenv(VirtualDiskId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disk.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disk.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disk.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disk.foo", "capacity"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualDiskName, os.Getenv(VirtualDiskName), os.Getenv("COMPUTE_VIRTUAL_MACHINE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disk.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disk.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disk.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_disk.foo", "capacity"),
				),
			},
			{
				Config:      testAccDataSourceVirtualDiskMissing,
				ExpectError: regexp.MustCompile("failed to find disk with id"),
			},
		},
	})
}

const testAccDataSourceVirtualDisk = `
data "cloudtemple_compute_virtual_disk" "foo" {
  id = "%s"
}
`

const testAccDataSourceVirtualDiskName = `
data "cloudtemple_compute_virtual_disk" "foo" {
  name               = "%s"
  virtual_machine_id = "%s"
}
`

const testAccDataSourceVirtualDiskMissing = `
data "cloudtemple_compute_virtual_disk" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
