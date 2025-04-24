package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceStorages(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceStorages,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des systèmes de stockage n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_storages.foo",
						"storages.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse storages count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected storages list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_storages.foo", "storages.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_storages.foo", "storages.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_storages.foo", "storages.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_storages.foo", "storages.0.site"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_storages.foo", "storages.0.resource_type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_storages.foo", "storages.0.host_address"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_storages.foo", "storages.0.is_ready"),
				),
			},
		},
	})
}

const testAccDataSourceStorages = `
data "cloudtemple_backup_storages" "foo" {}
`
