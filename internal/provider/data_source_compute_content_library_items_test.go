package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceLibraryItems(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceLibraryItems,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_items.foo", "content_library_id", "355b654d-6ea2-4773-80ee-246d3f56964f"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_items.foo", "content_library_items.#"),
				),
			},
		},
	})
}

const testAccDataSourceLibraryItems = `
data "cloudtemple_compute_content_library" "foo" {
  name = "PUBLIC"
}

data "cloudtemple_compute_content_library_items" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
}
`
