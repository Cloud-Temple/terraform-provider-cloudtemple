package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	// Catalogue ids of a DEV tenant, resolved out-of-band and injected via the
	// environment (the acceptance suite runs live against a real tenant). The AZ,
	// instance family and template ids reuse the constants already declared by the
	// catalogue datasource tests.
	PublicCloudVMInstanceBackupPolicyId = "PUBLIC_CLOUD_VM_BACKUP_POLICY_ID"
	PublicCloudVMInstanceNetworkId      = "PUBLIC_CLOUD_VM_NETWORK_ID"
)

// TestAccResourcePublicCloudVMInstance exercises the full lifecycle live:
// create (booted at creation via power_state = "on"), a read-back that populates
// the computed attributes, then import. cloud_init / os_network_adapter are not
// returned by the API (and are ForceNew), and power_state is derived from the
// status, so they are excluded from the import verification.
func TestAccResourcePublicCloudVMInstance(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourcePublicCloudVMInstanceConfig,
					os.Getenv(PublicCloudVMAvailabilityZoneId),
					os.Getenv(PublicCloudVMTemplateId),
					os.Getenv(PublicCloudVMInstanceFamilyId),
					os.Getenv(PublicCloudVMInstanceBackupPolicyId),
					os.Getenv(PublicCloudVMInstanceNetworkId),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_instance.test", "id"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_instance.test", "power_state", "on"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_instance.test", "status", "running"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_instance.test", "cpu", "1"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_instance.test", "memory", "2"),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_instance.test", "disks_size_gb"),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_instance.test", "availability_zone_name"),
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_instance.test", "template_name"),
				),
			},
			{
				ResourceName:      "cloudtemple_public_cloud_vm_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Not readable from the API (and ForceNew) / derived from status.
				ImportStateVerifyIgnore: []string{"os_network_adapter", "cloud_init", "power_state"},
			},
		},
	})
}

const testAccResourcePublicCloudVMInstanceConfig = `
resource "cloudtemple_public_cloud_vm_instance" "test" {
  name                 = "tf-acc-vm-instance"
  availability_zone_id = "%s"
  template_id          = "%s"
  instance_family_id   = "%s"
  cpu                  = 1
  memory               = 2
  backup_policy_id     = "%s"
  power_state          = "on"

  os_network_adapter {
    device_index = 0
    network_id   = "%s"
  }
}
`
