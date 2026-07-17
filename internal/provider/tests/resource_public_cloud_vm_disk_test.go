package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccResourcePublicCloudVMInstanceDisk creates a data disk on an existing VM
// (PUBLIC_CLOUD_VM_INSTANCE_ID), reads it back and imports it. Extend is exercised
// by the live E2E recipe (it requires the VM to be stopped).
func TestAccResourcePublicCloudVMInstanceDisk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourcePublicCloudVMDiskConfig, os.Getenv(PublicCloudVMInstanceId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_disk.data", "id"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_disk.data", "size", "10"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_disk.data", "is_primary", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_disk.data", "position"),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_disk.data", "storage_type"),
				),
			},
			{
				ResourceName:      "cloudtemple_public_cloud_vm_disk.data",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cloudtemple_public_cloud_vm_disk.data"]
					return rs.Primary.Attributes["virtual_machine_id"] + "/" + rs.Primary.ID, nil
				},
			},
		},
	})
}

const testAccResourcePublicCloudVMDiskConfig = `
resource "cloudtemple_public_cloud_vm_disk" "data" {
  virtual_machine_id = "%s"
  size               = 10
}
`
