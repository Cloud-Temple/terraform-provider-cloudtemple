package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpenIaaSPools(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpenIaaSPools,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des pools n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_pools.foo",
						"pools.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse pools count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected pools list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pools.foo", "pools.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pools.foo", "pools.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pools.foo", "pools.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pools.foo", "pools.0.machine_manager_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSPoolsWithFilter, os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des pools n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_pools.filtered",
						"pools.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse pools count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected pools list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pools.filtered", "pools.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pools.filtered", "pools.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pools.filtered", "pools.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_pools.filtered", "pools.0.machine_manager_id"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSPools = `
data "cloudtemple_compute_iaas_opensource_pools" "foo" {}
`

const testAccDataSourceOpenIaaSPoolsWithFilter = `
data "cloudtemple_compute_iaas_opensource_pools" "filtered" {
  machine_manager_id = "%s"
}
`
