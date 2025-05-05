package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VCenterId   = "COMPUTE_VCENTER_ID"
	VCenterName = "COMPUTE_VCENTER_NAME"
)

func TestAccDataSourceWorker(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceWorker, os.Getenv(VCenterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_manager.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_manager.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_manager.foo", "version"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_manager.foo", "vendor"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceWorkerName, os.Getenv(VCenterName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_manager.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_manager.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_manager.foo", "version"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_manager.foo", "vendor"),
				),
			},
			{
				Config:      testAccDataSourceWorkerMissing,
				ExpectError: regexp.MustCompile("failed to find worker with id"),
			},
		},
	})
}

const testAccDataSourceWorker = `
data "cloudtemple_compute_machine_manager" "foo" {
  id = "%s"
}
`

const testAccDataSourceWorkerName = `
data "cloudtemple_compute_machine_manager" "foo" {
  name = "%s"
}
`

const testAccDataSourceWorkerMissing = `
data "cloudtemple_compute_machine_manager" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
