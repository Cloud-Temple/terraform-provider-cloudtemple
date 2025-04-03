package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	VirtualControllerQty = "COMPUTE_VIRTUAL_CONTROLLER_QTY"
)

func TestAccDataSourceVirtualControllers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceVirtualControllers, os.Getenv(VirtualMachineId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_virtual_controllers.foo", "virtual_controllers.#", os.Getenv(VirtualControllerQty)),
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
