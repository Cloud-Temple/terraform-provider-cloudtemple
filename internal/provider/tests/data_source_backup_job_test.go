package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	JobId = "BACKUP_JOB_ID"
)

func TestAccDataSourceBackupJob(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceBackupJob, os.Getenv(JobId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "display_name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "status"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "policy_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceBackupJobName, "Hypervisor Inventory"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "display_name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "status"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job.foo", "policy_id"),
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
  id = "%s"
}
`

const testAccDataSourceBackupJobName = `
data "cloudtemple_backup_job" "foo" {
  name = "%s"
}
`

const testAccDataSourceBackupJobMissing = `
data "cloudtemple_backup_job" "foo" {
  id = "123456"
}
`
