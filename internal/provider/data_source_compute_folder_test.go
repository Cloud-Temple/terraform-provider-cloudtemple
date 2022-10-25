package provider

import (
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
		},
	})
}

const testAccDataSourceFolder = `
data "cloudtemple_compute_folder" "foo" {
  id = "b41ea9b1-4cca-44ed-9a76-2b598de03781"
}
`
