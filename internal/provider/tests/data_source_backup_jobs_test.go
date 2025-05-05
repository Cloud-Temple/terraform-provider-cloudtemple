package provider

import (
	"fmt"
	"strconv"
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
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des jobs n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_jobs.foo",
						"jobs.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse jobs count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected jobs list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_jobs.foo", "jobs.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_jobs.foo", "jobs.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_jobs.foo", "jobs.0.display_name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_jobs.foo", "jobs.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_jobs.foo", "jobs.0.status"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_jobs.foo", "jobs.0.policy_id"),
				),
			},
		},
	})
}

const testAccDataBackupJobs = `
data "cloudtemple_backup_jobs" "foo" {}
`
