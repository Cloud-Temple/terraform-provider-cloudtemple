package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceHostCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceHostCluster,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host_cluster.foo", "id", "dde72065-60f4-4577-836d-6ea074384d62"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host_cluster.foo", "name", "clu002-ucs01_FLO"),
				),
			},
			{
				Config:      testAccDataSourceHostClusterMissing,
				ExpectError: regexp.MustCompile("failed to find host cluster with id"),
			},
		},
	})
}

const testAccDataSourceHostCluster = `
data "cloudtemple_compute_host_cluster" "foo" {
  id = "dde72065-60f4-4577-836d-6ea074384d62"
}
`

const testAccDataSourceHostClusterMissing = `
data "cloudtemple_compute_host_cluster" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
