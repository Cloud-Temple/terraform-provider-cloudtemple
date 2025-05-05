package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpenIaaSBackups(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpenIaaSBackups,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des backups n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_iaas_opensource_backups.foo",
						"backups.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse backups count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected backups list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.foo", "backups.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.foo", "backups.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.foo", "backups.0.mode"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.foo", "backups.0.size"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.foo", "backups.0.timestamp"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSBackupsWithFilter, os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des backups n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_iaas_opensource_backups.filtered",
						"backups.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse backups count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected backups list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.filtered", "backups.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.filtered", "backups.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.filtered", "backups.0.mode"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.filtered", "backups.0.size"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backups.filtered", "backups.0.timestamp"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSBackups = `
data "cloudtemple_backup_iaas_opensource_backups" "foo" {}
`

const testAccDataSourceOpenIaaSBackupsWithFilter = `
data "cloudtemple_backup_iaas_opensource_backups" "filtered" {
  machine_manager_id = "%s"
}
`
