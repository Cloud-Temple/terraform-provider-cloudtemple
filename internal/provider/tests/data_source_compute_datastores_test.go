package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDatastores(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDatastores,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des datastores n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_datastores.foo",
						"datastores.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse datastores count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected datastores list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.max_capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.free_capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.virtual_machines_number"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastores.foo", "datastores.0.hosts_number"),
				),
			},
		},
	})
}

const testAccDataSourceDatastores = `
data "cloudtemple_compute_datastores" "foo" {}
`
