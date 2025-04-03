package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	DatastoreClusterId   = "COMPUTE_DATASTORE_CLUSTER_ID"
	DatastoreClusterName = "COMPUTE_DATASTORE_CLUSTER_NAME"
)

func TestAccDataSourceDatastoreCluster(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceDatastoreCluster, os.Getenv(DatastoreClusterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore_cluster.foo", "id", os.Getenv(DatastoreClusterId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore_cluster.foo", "name", os.Getenv(DatastoreClusterName)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceDatastoreClusterName, os.Getenv(DatastoreClusterName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore_cluster.foo", "id", os.Getenv(DatastoreClusterId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore_cluster.foo", "name", os.Getenv(DatastoreClusterName)),
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
  id = "%s"
}
`

const testAccDataSourceDatastoreClusterName = `
data "cloudtemple_compute_machine_manager" "vstack-01" {
	name = "vc-vstack-001-t0001"
}

data "cloudtemple_compute_datastore_cluster" "foo" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack-01.id
}
`

const testAccDataSourceDatastoreClusterMissing = `
data "cloudtemple_compute_datastore_cluster" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
