package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	DataCenterId = "DATACENTER_ID"
)

func TestAccResourceVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachine,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "datacenter_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "host_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "tags.environment"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "triggered_alarms.#"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineRelocate,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "datacenter_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "host_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "tags.environment"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "triggered_alarms.#"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachine,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				ResourceName:      "cloudtemple_compute_virtual_machine.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"datastore_cluster_id",
					"guest_operating_system_moref",
					"host_cluster_id",
					"extra_config",
				},
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineUpdate,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "memory"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "cpu"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "num_cores_per_socket"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "cpu_hot_add_enabled"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "cpu_hot_remove_enabled"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "memory_hot_add_enabled"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.foo", "tags.environment"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineRename,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform-rename"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachinePowerOn,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform-rename"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "power_state", "on"),
				),
			},
			{
				Destroy: true,
				Config: fmt.Sprintf(
					testAccResourceVirtualMachinePowerOn,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineClone,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "datacenter_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "host_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "datastore_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.cloned", "tags.environment"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineContentLibraryDeploy,
					os.Getenv(ContentLibraryName),
					os.Getenv(DataStoreName),
					os.Getenv(MachineManagerId),
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.content-library-deployed", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.content-library-deployed", "datacenter_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.content-library-deployed", "host_cluster_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.content-library-deployed", "tags.environment"),
				),
			},
		},
	})
}

const testAccResourceVirtualMachine = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  tags = {
		"environment" = "test"
  }
}
`

const testAccResourceVirtualMachineRelocate = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  tags = {
    "environment" = "test"
  }
}
`

const testAccResourceVirtualMachineUpdate = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  memory                 = 2 * 33554432
  cpu                    = 2
  num_cores_per_socket   = 2
  cpu_hot_add_enabled    = true
  cpu_hot_remove_enabled = true
  memory_hot_add_enabled = true

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  tags = {
		"environment" = "demo"
  }
}
`

const testAccResourceVirtualMachineRename = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-rename"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  lifecycle {
		prevent_destroy = true
  }
}
`

const testAccResourceVirtualMachinePowerOn = `
data "cloudtemple_backup_sla_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform-rename"
  power_state = "on"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"

  guest_operating_system_moref = "%s"

  backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.nobackup.id
  ]
}
`

const testAccResourceVirtualMachineClone = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

  tags = {
		"environment" = "test"
  }
}

resource "cloudtemple_compute_virtual_machine" "cloned" {
  name = "test-terraform-cloned"

  clone_virtual_machine_id     = cloudtemple_compute_virtual_machine.foo.id
  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"

  tags = {
		"environment" = "cloned"
  }
}
`

const testAccResourceVirtualMachineContentLibraryDeploy = `
data "cloudtemple_compute_content_library" "foo" {
  name = "%s"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  name               = "ubuntu-jammy-22.04-cloudimg"
}

data "cloudtemple_compute_datastore" "foo" {
	name = "%s"
	machine_manager_id = "%s"
}

resource "cloudtemple_compute_virtual_machine" "content-library-deployed" {
  name = "test-terraform-content-library-deployed"

  content_library_id      = data.cloudtemple_compute_content_library.foo.id
  content_library_item_id = data.cloudtemple_compute_content_library_item.foo.id

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_id          			 = data.cloudtemple_compute_datastore.foo.id

  tags = {
		"environment" = "deployed-from-content-library"
  }
}
`

const (
	// Substrate for the inline-VPC test below, gated on its own variables so the test
	// skips where it is not provisioned.
	//
	// Choosing it is not arbitrary. The only VMware create paths that produce network
	// adapters for an inline block to address are clone, content-library and
	// marketplace — the from-scratch create carries no network field at all. And the
	// clone must land on a host cluster whose hosts belong to the distributed switch
	// carrying the VPC portgroup, or the adapter patch fails with "Host <esx> is not a
	// member of VDS <dvs>". The reliable recipe, and the one this test was validated
	// with, is to clone a VM ALREADY on the target VPC network and reuse ITS
	// placement; pick one with a tiny disk, since the clone copies it.
	VMwareVPCCloneSourceId = "COMPUTE_VMWARE_VPC_CLONE_SOURCE_ID"
	VMwareVPCNetworkId     = "COMPUTE_VMWARE_VPC_NETWORK_ID"
	VMwareVPCStaticIP      = "COMPUTE_VMWARE_VPC_STATIC_IP"
)

func skipIfNoVMwareVPCEnv(t *testing.T) {
	for _, name := range []string{VMwareVPCCloneSourceId, VMwareVPCNetworkId, VMwareVPCStaticIP, DataCenterId, HostClusterId, DatastoreClusterId} {
		if os.Getenv(name) == "" {
			t.Skip(name + " not set (needs a clone source whose machine manager has a VPC-backed network, plus a FREE address in that VPC private network)")
		}
	}
}

// TestAccResourceVirtualMachineVPCInline exercises, live, the inline
// os_network_adapter `ip_address` on the VMware surface: a VM cloned from an
// existing one, whose inherited adapter is moved onto a VPC-backed network with a
// CHOSEN static IP, in a single apply.
//
// The clone path is used because the from-scratch create carries no network field
// at all (CreateVirtualMachineRequest has none) and therefore produces no adapter
// for an inline block to address — so it structurally cannot exercise this feature.
//
// Step 2 is the load-bearing one: `ip_address` is write-only, so a read path that
// failed to carry it across would leave a diff on every plan. PlanOnly turns that
// into a failure rather than a surprise.
func TestAccResourceVirtualMachineVPCInline(t *testing.T) {
	skipIfNoVMwareVPCEnv(t)
	config := fmt.Sprintf(
		testAccResourceVirtualMachineVPCInline,
		os.Getenv(VMwareVPCCloneSourceId),
		os.Getenv(DataCenterId),
		os.Getenv(HostClusterId),
		os.Getenv(DatastoreClusterId),
		os.Getenv(VMwareVPCNetworkId),
		os.Getenv(VMwareVPCStaticIP),
	)
	// Asserting the STATE alone would be complacent: os_network_adapter.0.ip_address
	// is overlaid from the configuration, so it would match even if the address had
	// never been pushed. The load-bearing assertion reads the LIVE registration by
	// MAC on the VPC plane, which is the only place the platform records it.
	c := vmwareVPCTestClient(t)
	wantIP := os.Getenv(VMwareVPCStaticIP)
	liveAddressMustMatch := func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["cloudtemple_compute_virtual_machine.vpc"]
		if !ok {
			return fmt.Errorf("resource not found in state")
		}
		mac := rs.Primary.Attributes["os_network_adapter.0.mac_address"]
		if mac == "" {
			return fmt.Errorf("os_network_adapter.0.mac_address is empty; cannot key the VPC registration")
		}
		sip, err := c.VPC().StaticIP().ReadByMAC(context.Background(), mac)
		if err != nil {
			return fmt.Errorf("ReadByMAC(%s): %w", mac, err)
		}
		if sip == nil {
			return fmt.Errorf("NO VPC static IP is registered for mac %s: the state claims %s but the platform has nothing — the address was never applied", mac, wantIP)
		}
		if sip.IPAddress != wantIP {
			return fmt.Errorf("the LIVE registration is %s, want the configured %s", sip.IPAddress, wantIP)
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_machine.vpc", "id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.vpc", "os_network_adapter.0.network_id", os.Getenv(VMwareVPCNetworkId)),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.vpc", "os_network_adapter.0.ip_address", wantIP),
					liveAddressMustMatch,
				),
			},
			{
				// No perpetual diff: the write-only value must survive a refresh.
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// vmwareVPCTestClient builds an API client for the out-of-band live assertion above.
// The acceptance suite already requires these credentials, so this adds no new
// configuration surface.
func vmwareVPCTestClient(t *testing.T) *client.Client {
	t.Helper()
	cfg := client.DefaultConfig()
	cfg.ClientID = os.Getenv(testClientIDEnvName)
	cfg.SecretID = os.Getenv(testSecretIDEnvName)
	c, err := client.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestAccResourceVirtualMachineIPOnNonVPCRejected proves live that a static IP on a
// non-VPC network is refused BEFORE anything is cloned. The platform silently
// discards such a value, so without the refusal it would be dead configuration the
// user believes took effect. It costs nothing: the refusal precedes the clone, so no
// disk is ever copied — which is why this one is NOT gated on the VPC substrate.
func TestAccResourceVirtualMachineIPOnNonVPCRejected(t *testing.T) {
	for _, name := range []string{VMwareVPCCloneSourceId, DataCenterId, HostClusterId, DatastoreClusterId, NetworkId} {
		if os.Getenv(name) == "" {
			t.Skip(name + " not set")
		}
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualMachineVPCInline,
					os.Getenv(VMwareVPCCloneSourceId),
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(NetworkId), // NOT VPC-backed
					"10.255.255.254",
				),
				ExpectError: regexp.MustCompile(`ip_address .* is not a VPC-backed private network`),
			},
		},
	})
}

const testAccResourceVirtualMachineVPCInline = `
resource "cloudtemple_compute_virtual_machine" "vpc" {
  name                     = "test-terraform-vpc-inline"
  clone_virtual_machine_id = "%s"

  datacenter_id        = "%s"
  host_cluster_id      = "%s"
  datastore_cluster_id = "%s"

  power_state = "off"

  os_network_adapter {
    network_id = "%s"
    ip_address = "%s"
  }

  tags = {
    created_by = "Terraform"
  }
}
`
