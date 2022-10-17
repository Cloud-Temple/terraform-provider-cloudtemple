package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceLibrary(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceLibrary,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "id", "355b654d-6ea2-4773-80ee-246d3f56964f"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "name", "PUBLIC"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "machine_manager_id", "9dba240e-a605-4103-bac7-5336d3ffd124"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "type", "SUBSCRIBED"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "datastore.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "datastore.0.id", "24371f16-b480-40d3-9587-82f97933abca"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "datastore.0.name", "ds002-bob-svc1-stor4-th3"),
				),
			},
			{
				Config: testAccDataSourceLibraryName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "id", "355b654d-6ea2-4773-80ee-246d3f56964f"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "name", "PUBLIC"),
				),
			},
			{
				Config:      testAccDataSourceLibraryMissing,
				ExpectError: regexp.MustCompile("failed to find content library with id"),
			},
		},
	})
}

const testAccDataSourceLibrary = `
data "cloudtemple_compute_content_library" "foo" {
  id = "355b654d-6ea2-4773-80ee-246d3f56964f"
}
`

const testAccDataSourceLibraryName = `
data "cloudtemple_compute_content_library" "foo" {
  name = "PUBLIC"
}
`

const testAccDataSourceLibraryMissing = `
data "cloudtemple_compute_content_library" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
