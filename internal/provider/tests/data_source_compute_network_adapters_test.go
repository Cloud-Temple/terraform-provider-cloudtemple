package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceNetworkAdapters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceNetworkAdapters, os.Getenv(VirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des adaptateurs réseau n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_network_adapters.foo",
						"network_adapters.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse network_adapters count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected network_adapters list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network_adapters.foo", "network_adapters.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network_adapters.foo", "network_adapters.0.virtual_machine_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network_adapters.foo", "network_adapters.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network_adapters.foo", "network_adapters.0.network_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network_adapters.foo", "network_adapters.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network_adapters.foo", "network_adapters.0.mac_address"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network_adapters.foo", "network_adapters.0.connected"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_network_adapters.foo", "network_adapters.0.auto_connect"),
				),
			},
			{
				Config: testAccDataSourceNetworkAdaptersMissing,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapters.foo", "network_adapters.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceNetworkAdapters = `
data "cloudtemple_compute_network_adapters" "foo" {
  virtual_machine_id = "%s"
}
`

const testAccDataSourceNetworkAdaptersMissing = `
data "cloudtemple_compute_network_adapters" "foo" {
  virtual_machine_id = "12345678-1234-5678-1234-567812345678"
}
`
