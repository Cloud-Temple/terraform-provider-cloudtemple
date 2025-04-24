package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSBackupPolicyId   = "BACKUP_IAAS_OPENSOURCE_POLICY_ID"
	OpenIaaSBackupPolicyName = "BACKUP_IAAS_OPENSOURCE_POLICY_NAME"
	OpenIaaSMachineManagerId = "COMPUTE_IAAS_OPENSOURCE_AVAILABILITY_ZONE_ID"
)

func TestAccDataSourceOpenIaaSBackupPolicy(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSBackupPolicy, os.Getenv(OpenIaaSBackupPolicyId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "machine_manager_name"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSBackupPolicyName, os.Getenv(OpenIaaSBackupPolicyName), os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policy.foo", "machine_manager_name"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSBackupPolicyMissing,
				ExpectError: regexp.MustCompile("failed to find backup policy with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSBackupPolicy = `
data "cloudtemple_backup_iaas_opensource_policy" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSBackupPolicyName = `
data "cloudtemple_backup_iaas_opensource_policy" "foo" {
  name               = "%s"
  machine_manager_id = "%s"
}
`

const testAccDataSourceOpenIaaSBackupPolicyMissing = `
data "cloudtemple_backup_iaas_opensource_policy" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
