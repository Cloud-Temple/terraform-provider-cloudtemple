package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceFolder,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_folder.foo", "id", "b41ea9b1-4cca-44ed-9a76-2b598de03781"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_folder.foo", "name", "Datacenters"),
				),
			},
			{
				Config:      testAccDataSourceFolderMissing,
				ExpectError: regexp.MustCompile("failed to find folder with id"),
			},
		},
	})
}

const testAccDataSourceFolder = `
data "cloudtemple_compute_folder" "foo" {
  id = "b41ea9b1-4cca-44ed-9a76-2b598de03781"
}
`

const testAccDataSourceFolderMissing = `
data "cloudtemple_compute_folder" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
