package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccResourcePublicCloudVMNetworkAdapter attaches a network adapter to an
// existing VM (PUBLIC_CLOUD_VM_INSTANCE_ID) on a network (PUBLIC_CLOUD_VM_NETWORK_ID),
// reads it back and imports it. Change-network and delete require the VM to be
// stopped and are exercised by the live E2E recipe. `ip_address` is write-only and
// is excluded from the import verification (it is never read back).
func TestAccResourcePublicCloudVMNetworkAdapter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourcePublicCloudVMNetworkAdapterConfig, os.Getenv(PublicCloudVMInstanceId), os.Getenv(PublicCloudVMNetworkId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_network_adapter.eth1", "id"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_network_adapter.eth1", "device_index", "1"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_network_adapter.eth1", "network_id", os.Getenv(PublicCloudVMNetworkId)),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_network_adapter.eth1", "type"),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_network_adapter.eth1", "provision_status"),
				),
			},
			{
				ResourceName:            "cloudtemple_public_cloud_vm_network_adapter.eth1",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"ip_address"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cloudtemple_public_cloud_vm_network_adapter.eth1"]
					return rs.Primary.Attributes["virtual_machine_id"] + "/" + rs.Primary.ID, nil
				},
			},
		},
	})
}

const testAccResourcePublicCloudVMNetworkAdapterConfig = `
resource "cloudtemple_public_cloud_vm_network_adapter" "eth1" {
  virtual_machine_id = "%s"
  device_index       = 1
  network_id         = "%s"
}
`
