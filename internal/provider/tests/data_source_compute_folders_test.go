package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceFolders(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceFolders,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des dossiers n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_folders.foo",
						"folders.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse folders count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected folders list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_folders.foo", "folders.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_folders.foo", "folders.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_folders.foo", "folders.0.machine_manager_id"),
				),
			},
		},
	})
}

const testAccDataSourceFolders = `
data "cloudtemple_compute_folders" "foo" {}
`
