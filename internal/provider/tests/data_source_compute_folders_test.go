package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	FolderQty = "COMPUTE_FOLDER_QTY"
)

func TestAccDataSourceFolders(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceFolders,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_folders.foo", "folders.#", os.Getenv(FolderQty)),
				),
			},
		},
	})
}

const testAccDataSourceFolders = `
data "cloudtemple_compute_folders" "foo" {}
`
