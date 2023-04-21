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
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "id", "10c6a0f7-076b-43aa-9230-bc975dcb1f30"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "name", "nobackup"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.type", "REPLICATION"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.use_encryption", "false"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.software", "true"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.site", "DC-TH3S"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.retention.0.age", "15"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.target.0.id", "1000"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.target.0.href", "https://10.12.8.1/api/site/1000"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.target.0.resource_type", "site"),
				),
			},
			{
				Config: testAccDataSourceBackupSLAPolicyName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "id", "10c6a0f7-076b-43aa-9230-bc975dcb1f30"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "name", "nobackup"),
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
  id = "10c6a0f7-076b-43aa-9230-bc975dcb1f30"
}
`

const testAccDataSourceBackupSLAPolicyName = `
data "cloudtemple_backup_sla_policy" "foo" {
  name = "nobackup"
}
`

const testAccDataSourceBackupSLAPolicyMissing = `
data "cloudtemple_backup_sla_policy" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
