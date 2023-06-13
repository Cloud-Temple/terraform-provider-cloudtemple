package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	MachineManagerName = "MACHINE_MANAGER_NAME"
	PolicyName2        = "BACKUP_METRICS_POLICY_NAME"
	PolicyId2          = "BACKUP_METRICS_POLICY_Id"
)

func TestAccResourceSLAPolicyAssignment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(
					testAccResourceSLAPolicyAssignment,
					os.Getenv(MachineManagerName),
					os.Getenv(VirtualDatacenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DatastoreClusterName),
					os.Getenv(OperatingSystemMoRef),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_backup_sla_policy_assignment.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_backup_sla_policy_assignment.foo", "sla_policy_ids.#", "2"),
					resource.TestCheckResourceAttr("cloudtemple_backup_sla_policy_assignment.foo", "sla_policy_ids.0", "4ea60b46-c2d4-4e5e-a5ac-b23c48dd3253"),
					resource.TestCheckResourceAttr("cloudtemple_backup_sla_policy_assignment.foo", "sla_policy_ids.1", "d7cbf073-a820-42cf-bf92-1d4a6570d568"),
				),
			},
			{
				Config: fmt.Sprintf(
					testAccResourceSLAPolicyAssignment,
					os.Getenv(MachineManagerName),
					os.Getenv(VirtualDatacenterName),
					os.Getenv(HostClusterName),
					os.Getenv(DatastoreClusterName),
					os.Getenv(OperatingSystemMoRef),
				),
				ResourceName:      "cloudtemple_backup_sla_policy_assignment.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccResourceSLAPolicyAssignment = `
data "cloudtemple_compute_machine_manager" "vstack" {
	name = "%s"
}

data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_host_cluster" "chc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

data "cloudtemple_compute_datastore_cluster" "cdc" {
  name = "%s"
  machine_manager_id = data.cloudtemple_compute_machine_manager.vstack.id
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-sla-policy"

  datacenter_id                = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.chc.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.cdc.id
  guest_operating_system_moref = "%s"
}

data "cloudtemple_backup_sla_policy" "test" {
  name = "test-fsn"
}

data "cloudtemple_backup_sla_policy" "weekly" {
  name = "sla001-weekly-th3s"
}


resource "cloudtemple_backup_sla_policy_assignment" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.foo.id
  sla_policy_ids = [
	data.cloudtemple_backup_sla_policy.weekly.id,
	data.cloudtemple_backup_sla_policy.test.id,
  ]
}
`
