package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	// Catalogue ids of a DEV tenant, resolved out-of-band and injected via the
	// environment (the acceptance suite runs live against a real tenant). The AZ,
	// instance family and image ids reuse the constants already declared by the
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
	skipIfNoPublicCloudVMImageEnv(t)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourcePublicCloudVMInstanceConfig,
					os.Getenv(PublicCloudVMAvailabilityZoneId),
					os.Getenv(PublicCloudVMImageId),
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
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_instance.test", "image_name"),
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
  image_id             = "%s"
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

const (
	// A VPC-backed network id on the test tenant, plus a free address inside its
	// VPC private network. Both are injected out-of-band: the acceptance suite runs
	// live, and hard-coding an address would collide with whatever the tenant
	// already has registered.
	PublicCloudVMVPCNetworkId = "PUBLIC_CLOUD_VM_VPC_NETWORK_ID"
	PublicCloudVMVPCStaticIP  = "PUBLIC_CLOUD_VM_VPC_STATIC_IP"
)

func skipIfNoPublicCloudVMVPCEnv(t *testing.T) {
	if os.Getenv(PublicCloudVMVPCNetworkId) == "" || os.Getenv(PublicCloudVMVPCStaticIP) == "" {
		t.Skip(PublicCloudVMVPCNetworkId + " / " + PublicCloudVMVPCStaticIP + " not set (set them to a VPC-backed network and a FREE address in its private network to run this test)")
	}
}

// TestAccResourcePublicCloudVMInstanceVPCInline proves live, end to end, that a
// VM can be created in ONE apply with its INLINE os_network_adapter attached to a
// VPC network and carrying a chosen static IP. This is the feature that the
// removed phase-1 guard used to forbid.
//
// The second step is the load-bearing one and is NOT decoration: `ip_address` is
// write-only (the platform exposes the registration only by MAC, on the VPC
// plane, so the VM read cannot re-derive it). A read path that dropped the value
// would leave a diff on every subsequent plan; `PlanOnly` turns that into a
// failure instead of a surprise for the user. It is what pins the absence of a
// perpetual diff.
func TestAccResourcePublicCloudVMInstanceVPCInline(t *testing.T) {
	skipIfNoPublicCloudVMImageEnv(t)
	skipIfNoPublicCloudVMVPCEnv(t)
	config := fmt.Sprintf(
		testAccResourcePublicCloudVMInstanceVPCConfig,
		os.Getenv(PublicCloudVMAvailabilityZoneId),
		os.Getenv(PublicCloudVMImageId),
		os.Getenv(PublicCloudVMInstanceFamilyId),
		os.Getenv(PublicCloudVMInstanceBackupPolicyId),
		os.Getenv(PublicCloudVMVPCNetworkId),
		os.Getenv(PublicCloudVMVPCStaticIP),
	)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_public_cloud_vm_instance.vpc", "id"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_instance.vpc", "status", "stopped"),
					// The VM is on the VPC network, with exactly the address asked for.
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_instance.vpc", "os_network_adapter.#", "1"),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_instance.vpc", "os_network_adapter.0.network_id", os.Getenv(PublicCloudVMVPCNetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_public_cloud_vm_instance.vpc", "os_network_adapter.0.ip_address", os.Getenv(PublicCloudVMVPCStaticIP)),
				),
			},
			{
				// No perpetual diff: re-planning the same config must be a no-op.
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccResourcePublicCloudVMInstanceIPOnNonVPCRejected proves live that the
// provider refuses a static IP on a non-VPC network. The platform accepts such a
// create and SILENTLY DISCARDS the address (measured on DEV), so without this
// refusal the user's chosen address would be dead configuration they believe took
// effect. ExpectError also proves the refusal happens at APPLY, before the VM
// exists — a create that succeeded and then errored would leave state behind and
// fail this step differently.
func TestAccResourcePublicCloudVMInstanceIPOnNonVPCRejected(t *testing.T) {
	skipIfNoPublicCloudVMImageEnv(t)
	if os.Getenv(PublicCloudVMInstanceNetworkId) == "" {
		t.Skip(PublicCloudVMInstanceNetworkId + " not set (set it to a PRIVATE BACKBONE network to run this test)")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourcePublicCloudVMInstanceVPCConfig,
					os.Getenv(PublicCloudVMAvailabilityZoneId),
					os.Getenv(PublicCloudVMImageId),
					os.Getenv(PublicCloudVMInstanceFamilyId),
					os.Getenv(PublicCloudVMInstanceBackupPolicyId),
					os.Getenv(PublicCloudVMInstanceNetworkId), // a Private Backbone network
					"10.255.255.254",
				),
				ExpectError: regexp.MustCompile(`ip_address .* is not a VPC network`),
			},
		},
	})
}

const testAccResourcePublicCloudVMInstanceVPCConfig = `
resource "cloudtemple_public_cloud_vm_instance" "vpc" {
  name                 = "tf-acc-vm-vpc-inline"
  availability_zone_id = "%s"
  image_id             = "%s"
  instance_family_id   = "%s"
  cpu                  = 1
  memory               = 2
  backup_policy_id     = "%s"
  power_state          = "off"

  os_network_adapter {
    device_index = 0
    network_id   = "%s"
    ip_address   = "%s"
  }
}
`
