package provider

import (
	"fmt"
	"os"
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
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_items.foo", "content_library_id", os.Getenv(ContentLibraryId)),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_items.foo", "content_library_items.#"),
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
