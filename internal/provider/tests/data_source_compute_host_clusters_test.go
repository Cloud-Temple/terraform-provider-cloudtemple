package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	HostClusterQty = "COMPUTE_HOST_CLUSTER_QTY"
)

func TestAccDataSourceHostClusters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceHostClusters,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_host_clusters.foo", "host_clusters.#", os.Getenv(HostClusterQty)),
				),
			},
		},
	})
}

const testAccDataSourceHostClusters = `
data "cloudtemple_compute_host_clusters" "foo" {}
`
