package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDatastoreClusters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDatastoreClusters,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.#", "2"),
				),
			},
		},
	})
}

const testAccDataSourceDatastoreClusters = `
data "cloudtemple_compute_datastore_clusters" "foo" {}
`
