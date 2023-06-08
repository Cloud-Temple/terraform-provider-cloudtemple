package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	JobsQty = "BACKUP_JOB_QTY"
)

func TestAccDataBackupJobs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataBackupJobs,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_jobs.foo", "jobs.#", os.Getenv(JobsQty)),
				),
			},
		},
	})
}

const testAccDataBackupJobs = `
data "cloudtemple_backup_jobs" "foo" {}
`
