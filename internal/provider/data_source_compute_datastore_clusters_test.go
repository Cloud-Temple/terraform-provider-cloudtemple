package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	DatastoreClustersQty = "COMPUTE_DATASTORE_CLUSTER_QTY"
)

func TestAccDataSourceDatastoreClusters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDatastoreClusters,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.#", os.Getenv(DatastoreClustersQty)),
				),
			},
		},
	})
}

const testAccDataSourceDatastoreClusters = `
data "cloudtemple_compute_datastore_clusters" "foo" {}
`
