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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host_cluster.foo", "id", "c80c4667-2f2d-4087-852b-995b0d5f1f2e"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host_cluster.foo", "name", "clu001-ucs12"),
				),
			},
			{
				Config: testAccDataSourceHostClusterName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host_cluster.foo", "id", "c80c4667-2f2d-4087-852b-995b0d5f1f2e"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host_cluster.foo", "name", "clu001-ucs12"),
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
  id = "c80c4667-2f2d-4087-852b-995b0d5f1f2e"
}
`

const testAccDataSourceHostClusterName = `
data "cloudtemple_compute_host_cluster" "foo" {
  name               = "clu001-ucs12"
	machine_manager_id = "8afdb4e8-b68d-4bb8-a606-3dc47cc2da0e"
}
`

const testAccDataSourceHostClusterMissing = `
data "cloudtemple_compute_host_cluster" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
