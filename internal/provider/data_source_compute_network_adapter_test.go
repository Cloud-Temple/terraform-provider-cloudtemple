package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	NetworkAdapterId   = "COMPUTE_NETWORK_ADAPTER_ID"
	NetworkAdapterName = "COMPUTE_NETWORK_ADAPTER_NAME"
)

func TestAccDataSourceNetworkAdapter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceNetworkAdapter, os.Getenv(NetworkAdapterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "id", os.Getenv(NetworkAdapterId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "name", os.Getenv(NetworkAdapterName)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "virtual_machine_id", os.Getenv(VirtualMachineIdAlternative)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceNetworkAdapterName, os.Getenv(NetworkAdapterName), os.Getenv(VirtualMachineIdAlternative)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "id", os.Getenv(NetworkAdapterId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_network_adapter.foo", "name", os.Getenv(NetworkAdapterName)),
				),
			},
			{
				Config:      testAccDataSourceNetworkAdapterMissing,
				ExpectError: regexp.MustCompile("failed to find network adapter with id"),
			},
		},
	})
}

const testAccDataSourceNetworkAdapter = `
data "cloudtemple_compute_network_adapter" "foo" {
  id = "%s"
}
`

const testAccDataSourceNetworkAdapterName = `
data "cloudtemple_compute_network_adapter" "foo" {
  name               = "%s"
  virtual_machine_id = "%s"
}
`

const testAccDataSourceNetworkAdapterMissing = `
data "cloudtemple_compute_network_adapter" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
