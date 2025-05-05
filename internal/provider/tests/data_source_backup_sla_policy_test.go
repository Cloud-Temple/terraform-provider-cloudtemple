package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	PolicyId   = "BACKUP_POLICY_ID"
	PolicyName = "BACKUP_POLICY_NAME"
)

func TestAccDataSourceBackupSLAPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceBackupSLAPolicy, os.Getenv(PolicyId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "sub_policies.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.site"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceBackupSLAPolicyName, os.Getenv(PolicyName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "sub_policies.#"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policy.foo", "sub_policies.0.site"),
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
  id = "%s"
}
`

const testAccDataSourceBackupSLAPolicyName = `
data "cloudtemple_backup_sla_policy" "foo" {
  name = "%s"
}
`

const testAccDataSourceBackupSLAPolicyMissing = `
data "cloudtemple_backup_sla_policy" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
