package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	FolderId   = "COMPUTE_FOLDER_ID"
	FolderName = "COMPUTE_FOLDER_NAME"
)

func TestAccDataSourceFolder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceFolder, os.Getenv(FolderId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_folder.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_folder.foo", "name"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceFolderName, os.Getenv(FolderName), os.Getenv(DataCenterId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_folder.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_folder.foo", "name"),
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
  id = "%s"
}
`

const testAccDataSourceFolderName = `
data "cloudtemple_compute_folder" "foo" {
  name = "%s"
	datacenter_id = "%s"
}
`

const testAccDataSourceFolderMissing = `
data "cloudtemple_compute_folder" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
