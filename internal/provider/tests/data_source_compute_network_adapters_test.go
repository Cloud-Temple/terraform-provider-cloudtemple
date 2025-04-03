package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	NetworkAdaptersQty = "COMPUTE_NETWORK_ADAPTER_QTY"
)

func TestAccDataSourceNetworkAdapters(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceNetworkAdapters, os.Getenv(VirtualMachineIdAlternative)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapters.foo", "network_adapters.#", os.Getenv(NetworkAdaptersQty)),
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
