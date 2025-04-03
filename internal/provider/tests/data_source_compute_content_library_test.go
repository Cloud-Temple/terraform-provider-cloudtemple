package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	MachineManagerId           = "COMPUTE_VCENTER_ID"
	ContentLibraryDatastoreQty = "COMPUTE_CONTENT_LIBRARY_DATASTORE_QTY"
)

func TestAccDataSourceLibrary(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceLibrary, os.Getenv(ContentLibraryId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "id", os.Getenv(ContentLibraryId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "name", os.Getenv(ContentLibraryName)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "machine_manager_id", os.Getenv(MachineManagerId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "type", os.Getenv(ContentLibraryType)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "datastore.#", os.Getenv(ContentLibraryDatastoreQty)),
					resource.TestCheckTypeSetElemNestedAttrs("data.cloudtemple_compute_content_library.foo", "datastore.*", map[string]string{
						"id":   os.Getenv(DataStoreId),
						"name": os.Getenv(DataStoreName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceLibraryName, os.Getenv(ContentLibraryName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "id", os.Getenv(ContentLibraryId)),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_content_library.foo", "name", os.Getenv(ContentLibraryName)),
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
  id = "%s"
}
`

const testAccDataSourceLibraryName = `
data "cloudtemple_compute_content_library" "foo" {
  name = "%s"
}
`

const testAccDataSourceLibraryMissing = `
data "cloudtemple_compute_content_library" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
