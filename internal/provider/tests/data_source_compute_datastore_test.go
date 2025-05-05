package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	DataStoreId   = "COMPUTE_DATASTORE_ID"
	DataStoreName = "COMPUTE_DATASTORE_NAME"
	DataStoresQty = "COMPUTE_DATASTORE_QTY"
)

func TestAccDataSourceDatastore(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceDatastore, os.Getenv(DataStoreId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "max_capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "free_capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "virtual_machines_number"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "hosts_number"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceDatastoreName, os.Getenv(DataStoreName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "moref"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "max_capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "free_capacity"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "virtual_machines_number"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_datastore.foo", "hosts_number"),
				),
			},
			{
				Config:      testAccDataSourceDatastoreMissing,
				ExpectError: regexp.MustCompile("failed to find datastore with id"),
			},
		},
	})
}

const testAccDataSourceDatastore = `
data "cloudtemple_compute_datastore" "foo" {
  id = "%s"
}
`

const testAccDataSourceDatastoreName = `
data "cloudtemple_compute_datastore" "foo" {
  name = "%s"
}
`

const testAccDataSourceDatastoreMissing = `
data "cloudtemple_compute_datastore" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
