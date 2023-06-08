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
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "id", os.Getenv(VirtualDiskId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "name", os.Getenv(VirtualDiskName)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "virtual_machine_id", os.Getenv(VirtualMachineIdAlternative)),
				),
			}, {
				Config: fmt.Sprintf(testAccDataSourceVirtualDiskName, os.Getenv(VirtualDiskName), os.Getenv(VirtualMachineIdAlternative)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "id", os.Getenv(VirtualDiskId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disk.foo", "name", os.Getenv(VirtualDiskName)),
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
