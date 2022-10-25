package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceWorker(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceWorker,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_worker.foo", "id", "9dba240e-a605-4103-bac7-5336d3ffd124"),
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
data "cloudtemple_compute_worker" "foo" {
  id = "9dba240e-a605-4103-bac7-5336d3ffd124"
}
`

const testAccDataSourceWorkerMissing = `
data "cloudtemple_compute_worker" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
