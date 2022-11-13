package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataBackupJobs(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataBackupJobs,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_jobs.foo", "jobs.#", "9"),
				),
			},
		},
	})
}

const testAccDataBackupJobs = `
data "cloudtemple_backup_jobs" "foo" {}
`
