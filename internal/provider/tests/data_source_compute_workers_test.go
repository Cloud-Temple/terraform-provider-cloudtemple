package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceWorkers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceWorkers,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des machine_managers n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_machine_managers.foo",
						"machine_managers.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse machine_managers count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected machine_managers count to be greater than 0, got %d", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_managers.foo", "machine_managers.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_managers.foo", "machine_managers.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_managers.foo", "machine_managers.0.version"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_machine_managers.foo", "machine_managers.0.vendor"),
				),
			},
		},
	})
}

const testAccDataSourceWorkers = `
data "cloudtemple_compute_machine_managers" "foo" {}
`
