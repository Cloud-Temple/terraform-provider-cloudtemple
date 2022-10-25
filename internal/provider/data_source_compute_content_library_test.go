package provider

import (
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
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "id", "355b654d-6ea2-4773-80ee-246d3f56964f"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "name", "PUBLIC"),
				),
			},
		},
	})
}

const testAccDataSourceLibrary = `
data "cloudtemple_compute_content_library" "foo" {
  id = "355b654d-6ea2-4773-80ee-246d3f56964f"
}
`
