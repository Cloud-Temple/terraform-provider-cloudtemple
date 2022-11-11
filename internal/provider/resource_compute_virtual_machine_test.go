package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceVirtualMachine(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceVirtualMachine,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "virtual_datacenter_id", "85d53d08-0fa9-491e-ab89-90919516df25"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "host_cluster_id", "dde72065-60f4-4577-836d-6ea074384d62"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "datastore_cluster_id", "6b06b226-ef55-4a0a-92bc-7aa071681b1b"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "guest_operating_system_moref", "amazonlinux2_64Guest"),
				),
			},
			{
				Config: testAccResourceVirtualMachinePowerOn,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "name", "test-terraform"),
					resource.TestCheckResourceAttr("cloudtemple_compute_virtual_machine.foo", "power_state", "on"),
				),
			},
			{
				// Trying to destroy a running VM will not work
				Destroy:     true,
				Config:      testAccResourceVirtualMachinePowerOn,
				ExpectError: regexp.MustCompile(`NOT_ALLOWED_IN_CURRENT_STATE`),
			},
			{
				// We stop the VM so that we can destroy it
				Config: testAccResourceVirtualMachine,
			},
		},
	})
}

const testAccResourceVirtualMachine = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform"

  virtual_datacenter_id        = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id              = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_cluster_id         = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
  guest_operating_system_moref = "amazonlinux2_64Guest"
}
`

const testAccResourceVirtualMachinePowerOn = `
resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform"
  power_state = "on"

  virtual_datacenter_id        = "85d53d08-0fa9-491e-ab89-90919516df25"
  host_cluster_id              = "dde72065-60f4-4577-836d-6ea074384d62"
  datastore_cluster_id         = "6b06b226-ef55-4a0a-92bc-7aa071681b1b"
  guest_operating_system_moref = "amazonlinux2_64Guest"
}
`
