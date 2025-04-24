package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualMachines(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualMachines,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des virtual_machines n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_virtual_machines.foo",
						"virtual_machines.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse virtual_machines count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected virtual_machines list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machines.foo", "virtual_machines.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machines.foo", "virtual_machines.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machines.foo", "virtual_machines.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machines.foo", "virtual_machines.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_machines.foo", "virtual_machines.0.power_state"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualMachines = `
data "cloudtemple_compute_virtual_machines" "foo" {}
`
