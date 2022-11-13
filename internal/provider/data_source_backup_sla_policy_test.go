package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupSLAPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSLAPolicy,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "id", "442718ef-44a1-43d7-9b57-2d910d74e928"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "name", "SLA_ADMIN"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.type", "REPLICATION"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.use_encryption", "false"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.software", "true"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.site", "DC-EQX6"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.retention.0.age", "15"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.trigger.0.frequency", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.trigger.0.type", "DAILY"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.trigger.0.activate_date", "1568617200000"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.target.0.id", "1000"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.target.0.href", "https://spp1-ctlabs-eqx6.backup.cloud-temple.lan/api/site/1000"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.target.0.resource_type", "site"),
				),
			},
			{
				Config: testAccDataSourceBackupSLAPolicyName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "id", "442718ef-44a1-43d7-9b57-2d910d74e928"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "name", "SLA_ADMIN"),
				),
			},
			{
				Config:      testAccDataSourceBackupSLAPolicyMissing,
				ExpectError: regexp.MustCompile("failed to find SLA policy with id"),
			},
		},
	})
}

const testAccDataSourceBackupSLAPolicy = `
data "cloudtemple_backup_sla_policy" "foo" {
  id = "442718ef-44a1-43d7-9b57-2d910d74e928"
}
`

const testAccDataSourceBackupSLAPolicyName = `
data "cloudtemple_backup_sla_policy" "foo" {
  name = "SLA_ADMIN"
}
`

const testAccDataSourceBackupSLAPolicyMissing = `
data "cloudtemple_backup_sla_policy" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
