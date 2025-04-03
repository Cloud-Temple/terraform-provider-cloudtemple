package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ContentLibrariesQty = "COMPUTE_CONTENT_LIBRARY_QTY"
)

func TestAccDataSourceLibraries(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceLibraries,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_libraries.foo", "content_libraries.#", os.Getenv(ContentLibrariesQty)),
				),
			},
		},
	})
}

const testAccDataSourceLibraries = `
data "cloudtemple_compute_content_libraries" "foo" {}
`
