package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceIaasOpensourceNetworkAdapter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceNetworkAdapter,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "network_id", os.Getenv("CLOUDTEMPLE_IAAS_OPENSOURCE_NETWORK_ID")),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "attached", "true"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "internal_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "mtu"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceNetworkAdapterDetached,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "network_id", os.Getenv("CLOUDTEMPLE_IAAS_OPENSOURCE_NETWORK_ID")),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "mac_address"),
					resource.TestCheckResourceAttr("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "attached", "false"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "internal_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "name"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_iaas_opensource_network_adapter.adapter", "mtu"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceIaasOpensourceNetworkAdapter,
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_TEMPLATE_ID"),
					os.Getenv("COMPUTE_IAAS_OPENSOURCE_NETWORK_ID"),
				),
				ResourceName:      "cloudtemple_compute_iaas_opensource_network_adapter.adapter",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccResourceIaasOpensourceNetworkAdapter = `
data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vm" {
  name        = "test-terraform-iaas-opensource-vm-for-network"
  template_id = "%s"
  cpu         = 2
  memory      = 2147483648
  power_state = "on"

	backup_sla_policies = [
		data.cloudtemple_backup_iaas_opensource_policy.nobackup.id
	]

	lifecycle {
		ignore_changes = [
			"memory",
		]
	}
}

resource "cloudtemple_compute_iaas_opensource_network_adapter" "adapter" {
  virtual_machine_id = cloudtemple_compute_iaas_opensource_virtual_machine.vm.id
  network_id         = "%s"
  attached           = true
}
`

const testAccResourceIaasOpensourceNetworkAdapterDetached = `
data "cloudtemple_backup_iaas_opensource_policy" "nobackup" {
	name = "nobackup"
}
	
resource "cloudtemple_compute_iaas_opensource_virtual_machine" "vm" {
  name        = "test-terraform-iaas-opensource-vm-for-network"
  template_id = "%s"
  cpu         = 2
  memory      = 2147483648
  power_state = "on"

	backup_sla_policies = [
		data.cloudtemple_backup_iaas_opensource_policy.nobackup.id
	]

	lifecycle {
		ignore_changes = [
			"memory",
		]
	}
}

resource "cloudtemple_compute_iaas_opensource_network_adapter" "adapter" {
  virtual_machine_id = cloudtemple_compute_iaas_opensource_virtual_machine.vm.id
  network_id         = "%s"
  attached           = false
}
`
