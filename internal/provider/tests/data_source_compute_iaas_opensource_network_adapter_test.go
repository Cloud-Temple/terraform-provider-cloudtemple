package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSNetworkAdapterId = "COMPUTE_IAAS_OPENSOURCE_NETWORK_ADAPTER_ID"
)

func TestAccDataSourceOpenIaaSNetworkAdapter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSNetworkAdapter, os.Getenv(OpenIaaSNetworkAdapterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapter.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapter.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapter.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapter.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network_adapter.foo", "network_id"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSNetworkAdapterMissing,
				ExpectError: regexp.MustCompile("failed to find network adapter with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSNetworkAdapter = `
data "cloudtemple_compute_iaas_opensource_network_adapter" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSNetworkAdapterMissing = `
data "cloudtemple_compute_iaas_opensource_network_adapter" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
