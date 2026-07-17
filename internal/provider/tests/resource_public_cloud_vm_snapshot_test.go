package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccResourcePublicCloudVMInstanceSnapshot creates a snapshot of an existing
// VM (PUBLIC_CLOUD_VM_INSTANCE_ID), reads it back, imports it and destroys it.
func TestAccResourcePublicCloudVMInstanceSnapshot(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourcePublicCloudVMSnapshotConfig, os.Getenv(PublicCloudVMInstanceId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_snapshot.test", "id"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_snapshot.test", "name", "tf-acc-snapshot"),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_snapshot.test", "status"),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_snapshot.test", "created_at"),
				),
			},
			{
				ResourceName:      "cloudtemple_public_cloud_vm_snapshot.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["cloudtemple_public_cloud_vm_snapshot.test"]
					return rs.Primary.Attributes["virtual_machine_id"] + "/" + rs.Primary.ID, nil
				},
			},
		},
	})
}

const testAccResourcePublicCloudVMSnapshotConfig = `
resource "cloudtemple_public_cloud_vm_snapshot" "test" {
  virtual_machine_id = "%s"
  name               = "tf-acc-snapshot"
}
`
