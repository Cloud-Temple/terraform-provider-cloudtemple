package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VistualDisksQty = "COMPUTE_VIRTUAL_DISK_QTY"
)

func TestAccDataSourceVirtualDisks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualDisks, os.Getenv(VirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.#", os.Getenv(VistualDisksQty)),
				),
			},
			{
				Config: testAccDataSourceVirtualDisksMissing,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_disks.foo", "virtual_disks.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualDisks = `
data "cloudtemple_compute_virtual_disks" "foo" {
  virtual_machine_id = "%s"
}
`

const testAccDataSourceVirtualDisksMissing = `
data "cloudtemple_compute_virtual_disks" "foo" {
  virtual_machine_id = "12345678-1234-5678-1234-567812345678"
}
`
