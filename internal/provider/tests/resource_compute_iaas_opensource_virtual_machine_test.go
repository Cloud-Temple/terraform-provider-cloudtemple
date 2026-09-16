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

// TestAccResourceIaasOpensourceVirtualMachineHostPlacementPoweredOffRejected
// pins the #355 fail-fast preflight wiring on a real apply: an explicitly
// configured host_id with power_state = "off" must be rejected before any
// resource is created (the preflight runs at the top of Create, before any
// client call). This is the call-site wiring proof that a CI unit test cannot
// produce, because GetRawConfig() is null in a unit-constructed ResourceData.
func TestAccResourceIaasOpensourceVirtualMachineHostPlacementPoweredOffRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachineHostOff,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
				),
				ExpectError: regexp.MustCompile(`host placement requires the VM to be running`),
			},
		},
	})
}

func TestAccResourceIaasOpensourceVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachine,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "name", "test-terraform-iaas-opensource-vm"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "cpu", "2"),
					//resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "memory", "2147483648"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "pool_id"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachineUpdate,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "name", "test-terraform-iaas-opensource-vm-updated"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "cpu", "4"),
					//resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "memory", "4294967296"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.foo", "pool_id"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachine,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
				),
				ResourceName:      "cloudtemple_compute_iaas_opensource_virtual_machine.foo",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"cloud_init",
					"template_id",
					"tools",
				},
			},
		},
	})
}

const testAccResourceIaasOpensourceVirtualMachine = `
data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_iaas_opensource_virtual_machine" "foo" {
  name        = "test-terraform-iaas-opensource-vm"
  template_id = "%s"
  cpu         = 2
  memory      = 2147483648
  power_state = "on"
  boot_firmware = "bios"
  auto_power_on = true

  tags = {
    "environment" = "test"
    "managed-by"  = "terraform"
  }

	backup_sla_policies = [
		data.cloudtemple_backup_iaas_opensource_policy.nobackup.id
	]

	lifecycle {
    ignore_changes = [
      memory,
    ]
  }
}
`

const testAccResourceIaasOpensourceVirtualMachineUpdate = `
data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_iaas_opensource_virtual_machine" "foo" {
  name        = "test-terraform-iaas-opensource-vm-updated"
  template_id = "%s"
  cpu         = 4
  memory      = 4294967296
  power_state = "on"
  boot_firmware = "bios"
  auto_power_on = true

  tags = {
    "environment" = "test"
    "managed-by"  = "terraform"
    "updated"     = "true"
  }

	backup_sla_policies = [
		data.cloudtemple_backup_iaas_opensource_policy.nobackup.id
	]

	lifecycle {
    ignore_changes = [
      memory,
    ]
  }
}
`

// testAccResourceIaasOpensourceVirtualMachineHostOff requests an explicit
// host_id while leaving the VM powered off. The host_id is a syntactically
// valid (IsUUID) but arbitrary UUID: the preflight rejects the combination
// before any API call, so the value is never sent to the platform. (#355)
const testAccResourceIaasOpensourceVirtualMachineHostOff = `
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "off_host" {
  name          = "test-terraform-iaas-opensource-vm-host-off"
  template_id   = "%s"
  cpu           = 2
  memory        = 2147483648
  power_state   = "off"
  host_id       = "11111111-1111-1111-1111-111111111111"
  boot_firmware = "bios"
}
`

const (
	// A template with EXACTLY TWO network adapters, a VPC-backed network id, and a
	// FREE address inside that VPC private network. Two adapters because the
	// resource requires one os_network_adapter block per template adapter, and the
	// DEV tenant has no single-adapter template.
	IaasOpensourceVPCTemplateId = "COMPUTE_IAAS_OPENSOURCE_VPC_TEMPLATE_ID"
	IaasOpensourceVPCNetworkId  = "COMPUTE_IAAS_OPENSOURCE_VPC_NETWORK_ID"
	IaasOpensourceVPCStaticIP   = "COMPUTE_IAAS_OPENSOURCE_VPC_STATIC_IP"
	// A SECOND, DIFFERENT network for the template's other adapter. Measured live:
	// the platform assigns an explicit address per (VM, network) pair and refuses
	// one address when several adapters share that network, so the two blocks must
	// target distinct networks.
	IaasOpensourceVPCSecondNetworkId = "COMPUTE_IAAS_OPENSOURCE_VPC_SECOND_NETWORK_ID"
	// A SECOND free address in the same VPC private network, needed to prove the
	// reconciliation is reachable when the state already claims the configured one.
	IaasOpensourceVPCSecondStaticIP = "COMPUTE_IAAS_OPENSOURCE_VPC_SECOND_STATIC_IP"
	// The "nobackup" SLA policy OF THE TEMPLATE'S MACHINE MANAGER. The tenant
	// refuses to power a VM on without an active SLA policy or "nobackup", and the
	// name is NOT unique across machine managers — resolving it by name can pick the
	// homonym of another manager, which the assign call then rejects with
	// "Could not assign all policies with the provided identifiers".
	IaasOpensourceNoBackupPolicyId = "COMPUTE_IAAS_OPENSOURCE_NOBACKUP_POLICY_ID"
)

func skipIfNoIaasOpensourceVPCEnv(t *testing.T) {
	for _, name := range []string{IaasOpensourceVPCTemplateId, IaasOpensourceVPCNetworkId, IaasOpensourceVPCStaticIP, IaasOpensourceVPCSecondNetworkId} {
		if os.Getenv(name) == "" {
			t.Skip(name + " not set (needs a 2-adapter template, a VPC-backed network and a FREE address in its private network)")
		}
	}
}

// TestAccResourceIaasOpensourceVirtualMachineVPCInline proves live that an
// OpenIaaS VM can be created, in ONE apply, with an inline os_network_adapter
// attached to a VPC network and carrying a CHOSEN static IP — the capability
// issue #376 was blocked on.
//
// Deliberately written without `lifecycle { ignore_changes = ... }`: the second
// step is a PlanOnly no-op check, which is the whole point. ip_address is
// write-only, so a read path that failed to carry it across would leave a diff on
// every plan; ignoring changes would hide exactly the defect this test exists to
// catch.
func TestAccResourceIaasOpensourceVirtualMachineVPCInline(t *testing.T) {
	skipIfNoIaasOpensourceVPCEnv(t)
	config := fmt.Sprintf(
		testAccResourceIaasOpensourceVirtualMachineVPC,
		os.Getenv(IaasOpensourceVPCTemplateId),
		os.Getenv(IaasOpensourceVPCNetworkId),
		os.Getenv(IaasOpensourceVPCStaticIP),
		os.Getenv(IaasOpensourceVPCSecondNetworkId),
	)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.#", "2"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.0.network_id", os.Getenv(IaasOpensourceVPCNetworkId)),
					// The chosen address must be recorded, and recorded on the RIGHT
					// adapter: index 1 declares none and must stay empty.
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.0.ip_address", os.Getenv(IaasOpensourceVPCStaticIP)),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.1.ip_address", ""),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.0.id"),
				),
			},
			{
				// No perpetual diff. This is what a write-only attribute risks, and
				// what the read-path carry-across exists to prevent.
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccResourceIaasOpensourceVirtualMachineIPOnNonVPCRejected proves live that
// a static IP on a non-VPC network is refused BEFORE the VM is created. The
// platform silently discards such a value, so without this refusal the address
// would be dead configuration the user believes took effect.
func TestAccResourceIaasOpensourceVirtualMachineIPOnNonVPCRejected(t *testing.T) {
	skipIfNoIaasOpensourceVPCEnv(t)
	if os.Getenv("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID") == "" {
		t.Skip("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID not set (needs a NON-VPC network)")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachineVPC,
					os.Getenv(IaasOpensourceVPCTemplateId),
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID"), // not VPC-backed
					"10.255.255.254",
					os.Getenv(IaasOpensourceVPCSecondNetworkId),
				),
				ExpectError: regexp.MustCompile(`ip_address .* is not a VPC-backed private network`),
			},
		},
	})
}

const testAccResourceIaasOpensourceVirtualMachineVPC = `
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vpc" {
  name        = "test-tf-iaas-opensource-vpc-inline"
  template_id = "%s"
  cpu         = 2
  memory      = 4294967296
  power_state = "off"

  os_network_adapter {
    network_id = "%s"
    ip_address = "%s"
  }

  os_network_adapter {
    network_id = "%s"
  }
}
`

// TestAccResourceIaasOpensourceVirtualMachineIPOnSharedNetworkRejected proves
// live that the provider refuses an explicit address when two declared adapters
// share the target network, BEFORE the create call. Without the preflight the
// platform refuses it too — but only after the create attempt, with the failure
// surfacing as an activity error rather than a plan-time-adjacent diagnostic.
func TestAccResourceIaasOpensourceVirtualMachineIPOnSharedNetworkRejected(t *testing.T) {
	skipIfNoIaasOpensourceVPCEnv(t)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachineVPC,
					os.Getenv(IaasOpensourceVPCTemplateId),
					os.Getenv(IaasOpensourceVPCNetworkId),
					os.Getenv(IaasOpensourceVPCStaticIP),
					os.Getenv(IaasOpensourceVPCNetworkId), // the SAME network: collision
				),
				ExpectError: regexp.MustCompile(`targets that same network`),
			},
		},
	})
}

// TestAccResourceIaasOpensourceVirtualMachineIPAddedOnUpdateRejected closes the
// bypass a create-only preflight would leave: the VM is first created with NO
// static IP, then a second step ADDS ip_address to the adapter that sits on a
// NON-VPC network.
//
// Without the update-path preflight that second apply would SUCCEED while doing
// nothing — a non-VPC adapter is never marked pending, so no call is made — and
// the read would then preserve the planned value, recording dead configuration as
// if it had been applied. Asserting only the create-time refusal would miss it
// entirely, which is why this two-step shape exists.
func TestAccResourceIaasOpensourceVirtualMachineIPAddedOnUpdateRejected(t *testing.T) {
	skipIfNoIaasOpensourceVPCEnv(t)
	nonVPC := os.Getenv("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID")
	if nonVPC == "" {
		t.Skip("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID not set (needs a NON-VPC network)")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				// Two adapters, distinct networks, NO static IP anywhere.
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachineVPCNoIP,
					os.Getenv(IaasOpensourceVPCTemplateId),
					os.Getenv(IaasOpensourceVPCNetworkId),
					nonVPC,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.#", "2"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.0.ip_address", ""),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.1.ip_address", ""),
				),
			},
			{
				// Now add a static IP to the adapter on the NON-VPC network.
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachineVPCSecondIP,
					os.Getenv(IaasOpensourceVPCTemplateId),
					os.Getenv(IaasOpensourceVPCNetworkId),
					nonVPC,
					"10.255.255.254",
				),
				ExpectError: regexp.MustCompile(`ip_address .* is not a VPC-backed private network`),
			},
		},
	})
}

const testAccResourceIaasOpensourceVirtualMachineVPCNoIP = `
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vpc" {
  name        = "test-tf-iaas-opensource-vpc-update"
  template_id = "%s"
  cpu         = 2
  memory      = 4294967296
  power_state = "off"

  os_network_adapter {
    network_id = "%s"
  }

  os_network_adapter {
    network_id = "%s"
  }
}
`

const testAccResourceIaasOpensourceVirtualMachineVPCSecondIP = `
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vpc" {
  name        = "test-tf-iaas-opensource-vpc-update"
  template_id = "%s"
  cpu         = 2
  memory      = 4294967296
  power_state = "off"

  os_network_adapter {
    network_id = "%s"
  }

  os_network_adapter {
    network_id = "%s"
    ip_address = "%s"
  }
}
`

// TestAccResourceIaasOpensourceVirtualMachineIPRejectedBeforeAnyMutation proves
// the placement of the update preflight, not merely its existence.
//
// Step 2 changes `name` AND adds an invalid `ip_address` in the SAME apply. The
// name PATCH sits earlier in the update function than the adapter reconciliation,
// so a preflight placed next to the adapter work would patch the name first and
// only then refuse — leaving the platform mutated by a configuration the provider
// was always going to reject.
//
// Step 3 is what makes the proof airtight: it re-plans the ORIGINAL config with
// PlanOnly. If the name had been patched platform-side, the refresh would surface
// it and the plan would be non-empty. An empty plan is positive evidence that
// NOTHING was mutated before the refusal.
func TestAccResourceIaasOpensourceVirtualMachineIPRejectedBeforeAnyMutation(t *testing.T) {
	skipIfNoIaasOpensourceVPCEnv(t)
	nonVPC := os.Getenv("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID")
	if nonVPC == "" {
		t.Skip("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID not set (needs a NON-VPC network)")
	}
	original := fmt.Sprintf(
		testAccResourceIaasOpensourceVirtualMachineVPCNoIPNamed,
		os.Getenv(IaasOpensourceVPCTemplateId),
		"test-tf-iaas-opensource-nomutation",
		os.Getenv(IaasOpensourceVPCNetworkId),
		nonVPC,
	)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: original,
				Check: resource.TestCheckResourceAttr(
					"cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "name", "test-tf-iaas-opensource-nomutation"),
			},
			{
				// Rename AND add an invalid static IP in one apply.
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceVirtualMachineVPCSecondIPNamed,
					os.Getenv(IaasOpensourceVPCTemplateId),
					"test-tf-iaas-opensource-nomutation-renamed",
					os.Getenv(IaasOpensourceVPCNetworkId),
					nonVPC,
					"10.255.255.254",
				),
				ExpectError: regexp.MustCompile(`ip_address .* is not a VPC-backed private network`),
			},
			{
				// The rename must NOT have happened: an empty plan proves it.
				Config:   original,
				PlanOnly: true,
			},
		},
	})
}

const testAccResourceIaasOpensourceVirtualMachineVPCNoIPNamed = `
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vpc" {
  name        = "%[2]s"
  template_id = "%[1]s"
  cpu         = 2
  memory      = 4294967296
  power_state = "off"

  os_network_adapter {
    network_id = "%[3]s"
  }

  os_network_adapter {
    network_id = "%[4]s"
  }
}
`

const testAccResourceIaasOpensourceVirtualMachineVPCSecondIPNamed = `
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vpc" {
  name        = "%[2]s"
  template_id = "%[1]s"
  cpu         = 2
  memory      = 4294967296
  power_state = "off"

  os_network_adapter {
    network_id = "%[3]s"
  }

  os_network_adapter {
    network_id = "%[4]s"
    ip_address = "%[5]s"
  }
}
`

// vpcTestClient builds an API client for the out-of-band assertions of the test
// below. The acceptance suite already requires these credentials (TestMain refuses
// to run without them), so this adds no new configuration surface.
func vpcTestClient(t *testing.T) *client.Client {
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

// TestAccResourceIaasOpensourceVirtualMachineIPReconvergesWithoutAdapterDiff is the
// REACHABILITY proof of the ip_address contract, and the reason the block-collection
// gate cannot be `d.HasChange("os_network_adapter")` alone.
//
// The hazard, reproduced here end to end rather than argued:
//
//	step 1  create a RUNNING VM with static IP A. A is registered live.
//	step 2  change the address to B **and** the cpu, with allow_vm_restart = false.
//	        The sizing refusal fires AFTER the ip preflight has passed, so the apply
//	        errors — but terraform-plugin-sdk/v2 persists the PLANNED values, so the
//	        state now claims B while the platform still has A.
//	step 3  allow the restart, changing nothing else. state ip == config ip == B, so
//	        there is NO os_network_adapter diff at all. With a HasChange-only gate the
//	        adapter blocks would not even be collected, the reconciliation would never
//	        run, and B would NEVER be applied — permanently, because a write-only
//	        attribute is not corrected by a refresh either.
//
// The final check reads the LIVE registration by MAC and requires B. An
// out-of-band deletion was tried first as a shortcut and is not possible: the
// platform refuses to delete a static IP whose source is `xoa`
// ("is not a custom static IP"), because the compute worker owns that registration.
func TestAccResourceIaasOpensourceVirtualMachineIPReconvergesWithoutAdapterDiff(t *testing.T) {
	skipIfNoIaasOpensourceVPCEnv(t)
	second := os.Getenv(IaasOpensourceVPCSecondStaticIP)
	if second == "" {
		t.Skip(IaasOpensourceVPCSecondStaticIP + " not set (this test needs a SECOND free address in the VPC private network)")
	}
	c := vpcTestClient(t)
	first := os.Getenv(IaasOpensourceVPCStaticIP)
	var mac string

	captureMAC := func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["cloudtemple_compute_iaas_opensource_virtual_machine.vpc"]
		if !ok {
			return fmt.Errorf("resource not found in state")
		}
		mac = rs.Primary.Attributes["os_network_adapter.0.mac_address"]
		if mac == "" {
			return fmt.Errorf("os_network_adapter.0.mac_address is empty; cannot key the VPC registration")
		}
		return nil
	}

	liveIPMustBe := func(want string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			sip, err := c.VPC().StaticIP().ReadByMAC(context.Background(), mac)
			if err != nil {
				return fmt.Errorf("ReadByMAC(%s): %w", mac, err)
			}
			if sip == nil {
				return fmt.Errorf("no static IP registered for mac %s, want %s", mac, want)
			}
			if sip.IPAddress != want {
				return fmt.Errorf("live static IP is %s, want %s — an apply with NO os_network_adapter diff failed to reach the reconciliation, so an optimistically recorded write-only ip_address would never be applied", sip.IPAddress, want)
			}
			return nil
		}
	}

	policy := os.Getenv(IaasOpensourceNoBackupPolicyId)
	if policy == "" {
		t.Skip(IaasOpensourceNoBackupPolicyId + " not set (the tenant refuses to power a VM on without an SLA policy; set it to the \"nobackup\" policy OF THE TEMPLATE'S machine manager)")
	}
	cfg := func(name, ip string, cpu int, allowRestart bool) string {
		return fmt.Sprintf(
			testAccResourceIaasOpensourceVirtualMachineVPCRunningIP,
			os.Getenv(IaasOpensourceVPCTemplateId), name, cpu, allowRestart,
			os.Getenv(IaasOpensourceVPCNetworkId), ip,
			os.Getenv(IaasOpensourceVPCSecondNetworkId), policy,
		)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg("test-tf-iaas-opensource-reconverge", first, 2, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "os_network_adapter.0.ip_address", first),
					captureMAC,
					liveIPMustBe(first),
				),
			},
			{
				// New address AND a sizing change, restart forbidden: the refusal
				// lands after the ip preflight, so the state records the new address
				// while the platform still holds the old one.
				Config:      cfg("test-tf-iaas-opensource-reconverge", second, 4, false),
				ExpectError: regexp.MustCompile(`needs to be powered off to apply changes to cpu, memory or num_cores_per_socket`),
			},
			{
				// Same address as the state now claims: NO adapter diff. Only the
				// widened collection gate can make the reconciliation happen.
				Config: cfg("test-tf-iaas-opensource-reconverge", second, 4, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.vpc", "cpu", "4"),
					liveIPMustBe(second),
				),
			},
		},
	})
}

const testAccResourceIaasOpensourceVirtualMachineVPCRunningIP = `
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vpc" {
  name             = "%[2]s"
  template_id      = "%[1]s"
  cpu              = %[3]d
  memory           = 4294967296
  power_state      = "on"
  allow_vm_restart = %[4]t
  # The template's guest has no PV drivers ready in a useful time on DEV; waiting
  # for them is irrelevant to what this test proves.
  wait_for_drivers_timeout = 0

  os_network_adapter {
    network_id = "%[5]s"
    ip_address = "%[6]s"
  }

  os_network_adapter {
    network_id = "%[7]s"
  }

  backup_sla_policies = ["%[8]s"]
}
`
