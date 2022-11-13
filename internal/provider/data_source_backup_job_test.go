package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupJob(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupJob,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "id", "1004"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "name", "Hypervisor Inventory"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "display_name", "Hypervisor Inventory"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "type", "catalog"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "status", "IDLE"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "policy_id", "1004"),
				),
			},
			{
				Config: testAccDataSourceBackupJobName,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "id", "1004"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "name", "Hypervisor Inventory"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "display_name", "Hypervisor Inventory"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "type", "catalog"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "status", "IDLE"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "policy_id", "1004"),
				),
			},
			{
				Config:      testAccDataSourceBackupJobMissing,
				ExpectError: regexp.MustCompile("failed to find job with id"),
			},
		},
	})
}

const testAccDataSourceBackupJob = `
data "cloudtemple_backup_job" "foo" {
  id = "1004"
}
`

const testAccDataSourceBackupJobName = `
data "cloudtemple_backup_job" "foo" {
  name = "Hypervisor Inventory"
}
`

const testAccDataSourceBackupJobMissing = `
data "cloudtemple_backup_job" "foo" {
  id = "123456"
}
`
