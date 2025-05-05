package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSVirtualMachineId = "COMPUTE_IAAS_OPENSOURCE_VIRTUAL_MACHINE_ID"
)

func TestAccDataSourceOpenIaaSNetworkAdapters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSNetworkAdapters, os.Getenv(OpenIaaSVirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des network_adapters n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_network_adapters.foo",
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
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapters.foo", "network_adapters.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapters.foo", "network_adapters.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapters.foo", "network_adapters.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapters.foo", "network_adapters.0.virtual_machine_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapters.foo", "network_adapters.0.network_id"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSNetworkAdapters = `
data "cloudtemple_compute_iaas_opensource_network_adapters" "foo" {
  virtual_machine_id = "%s"
}
`
