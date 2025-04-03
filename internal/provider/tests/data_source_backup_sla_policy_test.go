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
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "id", os.Getenv(PolicyId)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "name", os.Getenv(PolicyName)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceBackupSLAPolicyName, os.Getenv(PolicyName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "id", os.Getenv(PolicyId)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sla_policy.foo", "name", os.Getenv(PolicyName)),
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
