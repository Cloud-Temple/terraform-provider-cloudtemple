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
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "content_library_id", "355b654d-6ea2-4773-80ee-246d3f56964f"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "id", "8faded09-9f8b-4e27-a978-768f72f8e5f8"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "name", "20211115132417_master_linux-centos-8"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "description", "Centos 8"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "type", "ovf"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "creation_time", "2021-12-02T03:26:39Z"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "size", "1706045044"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "stored", "true"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_item.foo", "last_modified_time"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "ovf_properties.#", "0"),
				),
			},
			{
				Config: testAccDataSourceLibraryItemName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "id", "8faded09-9f8b-4e27-a978-768f72f8e5f8"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "name", "20211115132417_master_linux-centos-8"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "description", "Centos 8"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "type", "ovf"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "stored", "true"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_content_library_item.foo", "last_modified_time"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library_item.foo", "ovf_properties.#", "0"),
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
  name = "PUBLIC"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  id                 = "8faded09-9f8b-4e27-a978-768f72f8e5f8"
}
`

const testAccDataSourceLibraryItemName = `
data "cloudtemple_compute_content_library" "foo" {
  name = "PUBLIC"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  name               = "20211115132417_master_linux-centos-8"
}
`

const testAccDataSourceLibraryItemMissing = `
data "cloudtemple_compute_content_library" "foo" {
  name = "PUBLIC"
}

data "cloudtemple_compute_content_library_item" "foo" {
  content_library_id = data.cloudtemple_compute_content_library.foo.id
  id                 = "12345678-1234-5678-1234-567812345678"
}
`
