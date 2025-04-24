package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VirtualSwitchId   = "COMPUTE_VIRTUAL_SWITCH_ID"
	VirtualSwitchName = "COMPUTE_VIRTUAL_SWITCH_NAME"
)

func TestAccDataSourceVirtualSwitch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualSwitch, os.Getenv(VirtualSwitchId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switch.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switch.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switch.foo", "moref"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualSwitchName, os.Getenv(VirtualSwitchName), os.Getenv(MachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switch.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switch.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switch.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switch.foo", "folder_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switch.foo", "machine_manager_id"),
				),
			},
			{
				Config:      testAccDataSourceVirtualSwitchMissing,
				ExpectError: regexp.MustCompile("failed to find virtual switch with id"),
			},
		},
	})
}

const testAccDataSourceVirtualSwitch = `
data "cloudtemple_compute_virtual_switch" "foo" {
  id = "%s"
}
`

const testAccDataSourceVirtualSwitchName = `
data "cloudtemple_compute_virtual_switch" "foo" {
  name = "%s"
	machine_manager_id = "%s"
}
`

const testAccDataSourceVirtualSwitchMissing = `
data "cloudtemple_compute_virtual_switch" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
