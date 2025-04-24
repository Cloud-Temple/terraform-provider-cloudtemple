package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupSites(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSites,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des sites n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_sites.foo",
						"sites.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse sites count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected sites list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sites.foo", "sites.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sites.foo", "sites.0.name"),
				),
			},
		},
	})
}

const testAccDataSourceBackupSites = `
data "cloudtemple_backup_sites" "foo" {}
`
