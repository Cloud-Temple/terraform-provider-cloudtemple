package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataBackupJobSessions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataBackupJobSessions,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_job_sessions.foo", "job_sessions.#", "500"),
				),
			},
		},
	})
}

const testAccDataBackupJobSessions = `
data "cloudtemple_backup_job_sessions" "foo" {}
`
