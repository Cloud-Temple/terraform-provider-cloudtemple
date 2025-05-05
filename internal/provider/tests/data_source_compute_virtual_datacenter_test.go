package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VirtualDatacenterId   = "COMPUTE_VIRTUAL_DATACENTER_ID"
	VirtualDatacenterName = "COMPUTE_VIRTUAL_DATACENTER_NAME"
)

func TestAccDataSourceVirtualDatacenter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualDatacenter, os.Getenv(VirtualDatacenterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenter.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenter.foo", "name"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualDatacenterName, os.Getenv(VirtualDatacenterName), os.Getenv(MachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenter.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenter.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenter.foo", "tenant_id"),
				),
			},
			{
				Config:      testAccDataSourceVirtualDatacenterMissing,
				ExpectError: regexp.MustCompile("failed to find virtual datacenter with id"),
			},
		},
	})
}

const testAccDataSourceVirtualDatacenter = `
data "cloudtemple_compute_virtual_datacenter" "foo" {
  id = "%s"
}
`

const testAccDataSourceVirtualDatacenterName = `
data "cloudtemple_compute_virtual_datacenter" "foo" {
  name = "%s"
	machine_manager_id = "%s"
}
`

const testAccDataSourceVirtualDatacenterMissing = `
data "cloudtemple_compute_virtual_datacenter" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
