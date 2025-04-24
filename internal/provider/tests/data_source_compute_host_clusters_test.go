package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceHostClusters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceHostClusters,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des clusters d'hôtes n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_host_clusters.foo",
						"host_clusters.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse host_clusters count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected host_clusters list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_clusters.foo", "host_clusters.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_clusters.foo", "host_clusters.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_clusters.foo", "host_clusters.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_clusters.foo", "host_clusters.0.hosts.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_clusters.foo", "host_clusters.0.metrics.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_clusters.foo", "host_clusters.0.virtual_machines_number"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_clusters.foo", "host_clusters.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_host_clusters.foo", "host_clusters.0.datacenter_id"),
				),
			},
		},
	})
}

const testAccDataSourceHostClusters = `
data "cloudtemple_compute_host_clusters" "foo" {}
`
