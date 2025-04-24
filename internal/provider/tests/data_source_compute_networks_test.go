package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	NetworksQty = "COMPUTE_NETWORK_QTY"
)

func TestAccDataSourceNetworks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNetworks,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des réseaux n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_networks.foo",
						"networks.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse networks count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected networks list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_networks.foo", "networks.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_networks.foo", "networks.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_networks.foo", "networks.0.moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_networks.foo", "networks.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_networks.foo", "networks.0.virtual_machines_number"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_networks.foo", "networks.0.host_number"),
				),
			},
		},
	})
}

const testAccDataSourceNetworks = `
data "cloudtemple_compute_networks" "foo" {}
`
