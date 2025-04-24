package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSVirtualDiskId = "COMPUTE_IAAS_OPENSOURCE_VIRTUAL_DISK_ID"
)

func TestAccDataSourceOpenIaaSVirtualDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSVirtualDisk, os.Getenv(OpenIaaSVirtualDiskId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disk.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disk.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disk.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disk.foo", "storage_repository_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_virtual_disk.foo", "size"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSVirtualDiskMissing,
				ExpectError: regexp.MustCompile("failed to find virtual disk with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSVirtualDisk = `
data "cloudtemple_compute_iaas_opensource_virtual_disk" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSVirtualDiskMissing = `
data "cloudtemple_compute_iaas_opensource_virtual_disk" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
