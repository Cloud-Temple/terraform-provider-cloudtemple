package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	JobId         = "BACKUP_JOB_ID"
	JobName       = "BACKUP_JOB_NAME"
	JobDislayName = "BACKUP_JOB_DISPLAY_NAME"
	JobType       = "BACKUP_JOB_TYPE"
	JobStatus     = "BACKUP_JOB_STATUS"
	JobPolicyId   = "BACKUP_JOB_POLICY_ID"
)

func TestAccDataSourceBackupJob(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceBackupJob, os.Getenv(JobId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "id", os.Getenv(JobId)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "name", os.Getenv(JobName)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "display_name", os.Getenv(JobDislayName)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "type", os.Getenv(JobType)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "status", os.Getenv(JobStatus)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "policy_id", os.Getenv(JobPolicyId)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceBackupJobName, os.Getenv(JobName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "id", os.Getenv(JobId)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "name", os.Getenv(JobName)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "display_name", os.Getenv(JobDislayName)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "type", os.Getenv(JobType)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "status", os.Getenv(JobStatus)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job.foo", "policy_id", os.Getenv(JobPolicyId)),
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
