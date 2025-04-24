package provider

import (
	"fmt"
	"strconv"
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
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des sessions de job n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_job_sessions.foo",
						"job_sessions.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse job_sessions count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected job_sessions list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job_sessions.foo", "job_sessions.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job_sessions.foo", "job_sessions.0.job_name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job_sessions.foo", "job_sessions.0.job_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job_sessions.foo", "job_sessions.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job_sessions.foo", "job_sessions.0.status"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job_sessions.foo", "job_sessions.0.start"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_job_sessions.foo", "job_sessions.0.end"),
				),
			},
		},
	})
}

const testAccDataBackupJobSessions = `
data "cloudtemple_backup_job_sessions" "foo" {}
`
