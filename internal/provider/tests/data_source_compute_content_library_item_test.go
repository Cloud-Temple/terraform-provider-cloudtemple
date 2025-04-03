package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ContentLibraryId       = "COMPUTE_CONTENT_LIBRARY_ID"
	ContentLibraryName     = "COMPUTE_CONTENT_LIBRARY_NAME"
	ContentLibraryType     = "COMPUTE_CONTENT_LIBRARY_TYPE"
	ContentLibraryItemId   = "COMPUTE_CONTENT_LIBRARY_ITEM_ID"
	ContentLibraryItemName = "COMPUTE_CONTENT_LIBRARY_ITEM_NAME"
	ContentLibraryItemType = "COMPUTE_CONTENT_LIBRARY_ITEM_TYPE"
)

func TestAccDataSourceLibraryItem(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceLibraryItem, os.Getenv(ContentLibraryName), os.Getenv(ContentLibraryItemId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "content_library_id", os.Getenv(ContentLibraryId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "id", os.Getenv(ContentLibraryItemId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "name", os.Getenv(ContentLibraryItemName)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "type", os.Getenv(ContentLibraryItemType)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceLibraryItemName, os.Getenv(ContentLibraryName), os.Getenv(ContentLibraryItemName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "id", os.Getenv(ContentLibraryItemId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "name", os.Getenv(ContentLibraryItemName)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "type", os.Getenv(ContentLibraryItemType)),
				),
			},
			{
				Config:      fmt.Sprintf(testAccDataSourceLibraryItemMissing, os.Getenv(ContentLibraryName)),
				ExpectError: regexp.MustCompile("failed to find content library item with id"),
			},
		},
	})
}

const testAccDataSourceLibraryItem = `
data "cloudtemple_compute_content_library" "foo" {
  name = "%s"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  id                 = "%s"
}
`

const testAccDataSourceLibraryItemName = `
data "cloudtemple_compute_content_library" "foo" {
  name = "%s"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  name               = "%s"
}
`

const testAccDataSourceLibraryItemMissing = `
data "cloudtemple_compute_content_library" "foo" {
  name = "%s"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  id                 = "12345678-1234-5678-1234-567812345678"
}
`
