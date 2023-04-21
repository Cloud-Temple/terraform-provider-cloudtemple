package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceLibraryItem(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceLibraryItem,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "content_library_id", "25db620e-ffb1-4152-a786-b36041fe00c8"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "id", "8d34ec8a-488a-4757-84d0-ae7bc2c3659f"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "name", "runnable-ubuntu-template"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "description", "Template ubuntu"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "type", "vm-template"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "stored", "true"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_item.foo", "last_modified_time"),
				),
			},
			{
				Config: testAccDataSourceLibraryItemName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "id", "8d34ec8a-488a-4757-84d0-ae7bc2c3659f"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "name", "runnable-ubuntu-template"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "description", "Template ubuntu"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "type", "vm-template"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "stored", "true"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_item.foo", "last_modified_time"),
				),
			},
			{
				Config:      testAccDataSourceLibraryItemMissing,
				ExpectError: regexp.MustCompile("failed to find content library item with id"),
			},
		},
	})
}

const testAccDataSourceLibraryItem = `
data "cloudtemple_compute_content_library" "foo" {
  name = "local-vc-vstack-001-t0001"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  id                 = "8d34ec8a-488a-4757-84d0-ae7bc2c3659f"
}
`

const testAccDataSourceLibraryItemName = `
data "cloudtemple_compute_content_library" "foo" {
  name = "local-vc-vstack-001-t0001"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  name               = "runnable-ubuntu-template"
}
`

const testAccDataSourceLibraryItemMissing = `
data "cloudtemple_compute_content_library" "foo" {
  name = "local-vc-vstack-001-t0001"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  id                 = "12345678-1234-5678-1234-567812345678"
}
`
