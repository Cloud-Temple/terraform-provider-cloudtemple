package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ResourcePoolQty = "COMPTE_RESOURCE_POOL_QTY"
)

func TestAccDataSourceResourcePools(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceResourcePools,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des pools de ressources n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_resource_pools.foo",
						"resource_pools.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse resource_pools count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected resource_pools list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pools.foo", "resource_pools.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pools.foo", "resource_pools.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pools.foo", "resource_pools.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pools.foo", "resource_pools.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pools.foo", "resource_pools.0.parent.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_resource_pools.foo", "resource_pools.0.parent.0.type"),
				),
			},
		},
	})
}

const testAccDataSourceResourcePools = `
data "cloudtemple_compute_resource_pools" "foo" {}
`
