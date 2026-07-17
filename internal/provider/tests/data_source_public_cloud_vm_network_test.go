package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PublicCloudVMNetworkId   = "PUBLIC_CLOUD_VM_NETWORK_ID"
	PublicCloudVMNetworkName = "PUBLIC_CLOUD_VM_NETWORK_NAME"
)

func TestAccDataSourcePublicCloudVMNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMNetwork, os.Getenv(PublicCloudVMNetworkId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_network.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_network.foo", "name"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourcePublicCloudVMNetworkName, os.Getenv(PublicCloudVMNetworkName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_public_cloud_vm_network.foo", "id"),
					resource.TestCheckResourceAttr("data.cloudtemple_public_cloud_vm_network.foo", "name", os.Getenv(PublicCloudVMNetworkName)),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourcePublicCloudVMNetworkConflict, os.Getenv(PublicCloudVMNetworkId), os.Getenv(PublicCloudVMNetworkName)),
				ExpectError: regexp.MustCompile(`"id": conflicts with name`),
			},
			{
				// An unknown network id does not return a clean 404 (the platform
				// returns 400/500); the read fails closed with the backend error.
				Config:      testAccDataSourcePublicCloudVMNetworkMissing,
				ExpectError: regexp.MustCompile(`(?i)(unexpected response code|failed to find network)`),
			},
		},
	})
}

const testAccDataSourcePublicCloudVMNetwork = `
data "cloudtemple_public_cloud_vm_network" "foo" {
  id = "%s"
}
`

const testAccDataSourcePublicCloudVMNetworkName = `
data "cloudtemple_public_cloud_vm_network" "foo" {
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMNetworkConflict = `
data "cloudtemple_public_cloud_vm_network" "foo" {
  id   = "%s"
  name = "%s"
}
`

const testAccDataSourcePublicCloudVMNetworkMissing = `
data "cloudtemple_public_cloud_vm_network" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
