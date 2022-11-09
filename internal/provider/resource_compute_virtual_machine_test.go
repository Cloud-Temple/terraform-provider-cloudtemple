package provider

import (
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
