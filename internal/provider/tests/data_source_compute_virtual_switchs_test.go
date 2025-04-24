package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VirtualSwitchQty = "COMPUTE_VIRTUAL_SWITCH_QTY"
)

func TestAccDataSourceVirtualSwitchs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualSwitchs,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des virtual_switchs n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_virtual_switchs.foo",
						"virtual_switchs.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse virtual_switchs count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected virtual_switchs list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switchs.foo", "virtual_switchs.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switchs.foo", "virtual_switchs.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switchs.foo", "virtual_switchs.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_switchs.foo", "virtual_switchs.0.machine_manager_id"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualSwitchs = `
data "cloudtemple_compute_virtual_switchs" "foo" {}
`
