package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VirtualDatacenterQty = "COMPUTE_VIRTUAL_DATACENTER_QTY"
)

func TestAccDataSourceVirtualDatacenters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceVirtualDatacenters,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des datacenters virtuels n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_virtual_datacenters.foo",
						"virtual_datacenters.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse virtual_datacenters count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected virtual_datacenters list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenters.foo", "virtual_datacenters.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenters.foo", "virtual_datacenters.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenters.foo", "virtual_datacenters.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_datacenters.foo", "virtual_datacenters.0.tenant_id"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualDatacenters = `
data "cloudtemple_compute_virtual_datacenters" "foo" {}
`
