package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceSLAPolicyAssignment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSLAPolicyAssignment,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("cloudtemple_backup_sla_policy_assignment.foo", "virtual_machine_id"),
					resource.TestCheckResourceAttr("cloudtemple_backup_sla_policy_assignment.foo", "sla_policy_ids.#", "2"),
					resource.TestCheckResourceAttr("cloudtemple_backup_sla_policy_assignment.foo", "sla_policy_ids.0", "442718ef-44a1-43d7-9b57-2d910d74e928"),
					resource.TestCheckResourceAttr("cloudtemple_backup_sla_policy_assignment.foo", "sla_policy_ids.1", "a90e8505-1a82-4878-9410-0912ec63fec3"),
				),
			},
			{
				Config:            testAccResourceSLAPolicyAssignment,
				ResourceName:      "cloudtemple_backup_sla_policy_assignment.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccResourceSLAPolicyAssignment = `
data "cloudtemple_compute_virtual_datacenter" "dc" {
  name = "DC-EQX6"
}

data "cloudtemple_compute_host_cluster" "flo" {
  name = "clu002-ucs01_FLO"
}

data "cloudtemple_compute_datastore_cluster" "koukou" {
  name = "sdrs001-LIVE_KOUKOU"
}

resource "cloudtemple_compute_virtual_machine" "foo" {
  name = "test-terraform-sla-policy"

  virtual_datacenter_id        = data.cloudtemple_compute_virtual_datacenter.dc.id
  host_cluster_id              = data.cloudtemple_compute_host_cluster.flo.id
  datastore_cluster_id         = data.cloudtemple_compute_datastore_cluster.koukou.id
  guest_operating_system_moref = "amazonlinux2_64Guest"
}

data "cloudtemple_backup_sla_policy" "admin" {
  name = "SLA_ADMIN"
}

data "cloudtemple_backup_sla_policy" "daily" {
  name = "SLA_DAILY"
}

resource "cloudtemple_backup_sla_policy_assignment" "foo" {
  virtual_machine_id = cloudtemple_compute_virtual_machine.foo.id
  sla_policy_ids = [
	data.cloudtemple_backup_sla_policy.daily.id,
	data.cloudtemple_backup_sla_policy.admin.id,
  ]
}
`
