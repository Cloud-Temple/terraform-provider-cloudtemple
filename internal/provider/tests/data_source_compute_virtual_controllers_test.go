package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceVirtualControllers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualControllers, os.Getenv(VirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des contrôleurs virtuels n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_virtual_controllers.foo",
						"virtual_controllers.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse virtual_controllers count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected virtual_controllers list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_controllers.foo", "virtual_controllers.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_controllers.foo", "virtual_controllers.0.virtual_machine_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_controllers.foo", "virtual_controllers.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_controllers.foo", "virtual_controllers.0.label"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_virtual_controllers.foo", "virtual_controllers.0.hot_add_remove"),
				),
			},
			{
				Config: testAccDataSourceVirtualControllersMissing,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_controllers.foo", "virtual_controllers.#", "0"),
				),
			},
		},
	})
}

const testAccDataSourceVirtualControllers = `
data "cloudtemple_compute_virtual_controllers" "foo" {
  virtual_machine_id = "%s"
}
`

const testAccDataSourceVirtualControllersMissing = `
data "cloudtemple_compute_virtual_controllers" "foo" {
  virtual_machine_id = "12345678-1234-5678-1234-567812345678"
}
`
