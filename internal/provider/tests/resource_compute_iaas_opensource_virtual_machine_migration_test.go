package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccResourceIaasOpensourceVirtualMachineHostMigration is the end-to-end
// proof of #355: changing host_id on a RUNNING VM must live-migrate it to the
// requested host (same pool) and converge — never a silent no-op.
//
// It REQUIRES two hosts in the SAME XCP-ng pool/cluster, provided via:
//   - COMPUTE_IAAS_OPENSOURCE_HOST_A_ID  (initial host)
//   - COMPUTE_IAAS_OPENSOURCE_HOST_B_ID  (migration target, same pool as A)
//   - COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID
//
// It is SKIPPED when the two host ids are not set, so it does not run on a
// single-host environment (as of 2026-06, the recette has a single host, so
// this scenario is prepared but not yet runnable).
//
// Why this is a genuine #355 regression test (not complacent):
//   - Step 1 boots the VM on host A and pins host_id == A.
//   - Step 2 changes host_id to B. With the fix, the provider relocates the VM,
//     waits for the activity, and the post-apply refresh reads host_id == B —
//     so the assertion passes AND the implicit post-step plan is empty.
//     WITHOUT the fix (the original bug), the apply is a silent no-op: the VM
//     stays on A, the refresh writes host_id == A, the TestCheck for B fails,
//     and the post-step plan is non-empty (the same host_id diff reappears) —
//     resource.Test fails on both counts. So this step goes RED without the fix.
func TestAccResourceIaasOpensourceVirtualMachineHostMigration(t *testing.T) {
	hostA := os.Getenv("COMPUTE_IAAS_OPENSOURCE_HOST_A_ID")
	hostB := os.Getenv("COMPUTE_IAAS_OPENSOURCE_HOST_B_ID")
	if hostA == "" || hostB == "" {
		t.Skip("requires two hosts in the same pool: set COMPUTE_IAAS_OPENSOURCE_HOST_A_ID and COMPUTE_IAAS_OPENSOURCE_HOST_B_ID (intra-pool migration, #355)")
	}

	template := os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID")
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				// Boot the VM on host A.
				Config: fmt.Sprintf(testAccResourceIaasOpensourceVirtualMachineOnHost, template, hostA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.mig", "host_id", hostA),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.mig", "power_state", "on"),
				),
			},
			{
				// Migrate the running VM to host B (intra-pool). The fix must
				// actually move it and converge; the post-step plan must be empty.
				Config: fmt.Sprintf(testAccResourceIaasOpensourceVirtualMachineOnHost, template, hostB),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.mig", "host_id", hostB),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_virtual_machine.mig", "power_state", "on"),
				),
			},
		},
	})
}

// testAccResourceIaasOpensourceVirtualMachineOnHost pins the VM to a given host
// while running. %s = template id, %s = host id.
const testAccResourceIaasOpensourceVirtualMachineOnHost = `
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "mig" {
  name          = "test-terraform-iaas-opensource-vm-migration"
  template_id   = "%s"
  cpu           = 2
  memory        = 2147483648
  power_state   = "on"
  host_id       = "%s"
  boot_firmware = "bios"
  auto_power_on = true

  lifecycle {
    ignore_changes = [
      memory,
    ]
  }
}
`
