package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceLibraries(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceLibraries,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des bibliothèques de contenu n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_content_libraries.foo",
						"content_libraries.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse content_libraries count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected content_libraries list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_libraries.foo", "content_libraries.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_libraries.foo", "content_libraries.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_libraries.foo", "content_libraries.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_libraries.foo", "content_libraries.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_libraries.foo", "content_libraries.0.datastore.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_libraries.foo", "content_libraries.0.datastore.0.name"),
				),
			},
		},
	})
}

const testAccDataSourceLibraries = `
data "cloudtemple_compute_content_libraries" "foo" {}
`
