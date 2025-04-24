package provider

import (
	"fmt"
	"strconv"
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
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des clusters de datastores n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_datastore_clusters.foo",
						"datastore_clusters.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse datastore_clusters count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected datastore_clusters list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.datacenter_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.datastores.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.metrics.0.free_capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.metrics.0.max_capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore_clusters.foo", "datastore_clusters.0.metrics.0.enabled"),
				),
			},
		},
	})
}

const testAccDataSourceDatastoreClusters = `
data "cloudtemple_compute_datastore_clusters" "foo" {}
`
