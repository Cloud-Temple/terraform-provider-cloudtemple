package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDatastoreCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDatastoreCluster,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore_cluster.foo", "id", "6b06b226-ef55-4a0a-92bc-7aa071681b1b"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore_cluster.foo", "name", "sdrs001-LIVE_KOUKOU"),
				),
			},
			{
				Config:      testAccDataSourceDatastoreClusterMissing,
				ExpectError: regexp.MustCompile("failed to find datastore cluster with id"),
			},
		},
	})
}

const testAccDataSourceDatastoreCluster = `
data "cloudtemple_compute_datastore_cluster" "foo" {
  id = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
}
`

const testAccDataSourceDatastoreClusterMissing = `
data "cloudtemple_compute_datastore_cluster" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
