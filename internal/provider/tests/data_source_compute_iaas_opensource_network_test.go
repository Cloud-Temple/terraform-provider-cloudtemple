package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSNetworkId   = "COMPUTE_IAAS_OPENSOURCE_NETWORK_ID"
	OpenIaaSNetworkName = "COMPUTE_IAAS_OPENSOURCE_NETWORK_NAME"
)

func TestAccDataSourceOpenIaaSNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSNetwork, os.Getenv(OpenIaaSNetworkId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network.foo", "machine_manager_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSNetworkName, os.Getenv(OpenIaaSNetworkName), os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_network.foo", "machine_manager_id"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSNetworkMissing,
				ExpectError: regexp.MustCompile("failed to find network with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSNetwork = `
data "cloudtemple_compute_iaas_opensource_network" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSNetworkName = `
data "cloudtemple_compute_iaas_opensource_network" "foo" {
  name               = "%s"
  machine_manager_id = "%s"
}
`

const testAccDataSourceOpenIaaSNetworkMissing = `
data "cloudtemple_compute_iaas_opensource_network" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
