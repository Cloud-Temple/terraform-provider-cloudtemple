package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceVirtualController(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualControllerContentLib,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_controller.bar", "id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_controller.bar", "label"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_controller.bar", "type"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_controller.bar", "virtual_machine_id"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceVirtualController,
					os.Getenv(DataCenterId),
					os.Getenv(HostClusterId),
					os.Getenv(DatastoreClusterId),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_controller.baz", "id"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_controller.baz", "label"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_controller.baz", "type"),
					resource.TestCheckResourceAttrSet("cloudtemple_compute_virtual_controller.baz", "virtual_machine_id"),
				),
			},
		},
	})
}

const testAccResourceVirtualControllerContentLib = `
data "cloudtemple_compute_content_library_item" "foo" {
	id = "cfccc9e9-ca6d-4201-b302-73967d0cf9ef"
	content_library_id = "77f4226d-f05d-49db-a544-9333f96e65a5"
}

data "cloudtemple_backup_sla_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform-controller"
  power_state = "on"

  memory               = 8 * 1024 * 1024 * 1024
  cpu                  = 4
  num_cores_per_socket = 1

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"

	backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.nobackup.id
	]
}

resource "cloudtemple_compute_virtual_controller" "bar" {
  virtual_machine_id      = cloudtemple_compute_virtual_machine.foo.id
  type                    = "CD/DVD"
  content_library_item_id = data.cloudtemple_compute_content_library_item.foo.id
  connected               = true
  mounted                 = true
}
`

const testAccResourceVirtualController = `
data "cloudtemple_backup_sla_policy" "nobackup" {
	name = "nobackup"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name        = "test-terraform-controller"
  power_state = "off"

  memory               = 8 * 1024 * 1024 * 1024
  cpu                  = 4
  num_cores_per_socket = 1

  datacenter_id                = "%s"
  host_cluster_id              = "%s"
  datastore_cluster_id         = "%s"
  guest_operating_system_moref = "%s"
	
	backup_sla_policies = [
		data.cloudtemple_backup_sla_policy.nobackup.id
	]
}

resource "cloudtemple_compute_virtual_controller" "baz" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.foo.id
  type               = "SCSI"
  sub_type           = "ParaVirtual"
}
`
