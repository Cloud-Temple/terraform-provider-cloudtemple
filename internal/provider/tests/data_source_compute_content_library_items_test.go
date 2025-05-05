package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceLibraryItems(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceLibraryItems, os.Getenv(ContentLibraryName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_items.foo", "content_library_id"),
					// Vérifier que la liste des éléments de la bibliothèque de contenu n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_content_library_items.foo",
						"content_library_items.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse content_library_items count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected content_library_items list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_items.foo", "content_library_items.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_items.foo", "content_library_items.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_items.foo", "content_library_items.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_items.foo", "content_library_items.0.creation_time"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_items.foo", "content_library_items.0.size"),
				),
			},
		},
	})
}

const testAccDataSourceLibraryItems = `
data "cloudtemple_compute_content_library" "foo" {
  name = "%s"
}

data "cloudtemple_compute_content_library_items" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
}
`
