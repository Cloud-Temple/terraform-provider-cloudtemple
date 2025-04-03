package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	StoragesQty = "BACKUP_STORAGE_QTY"
)

func TestAccDataSourceStorages(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceStorages,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_storages.foo", "storages.#", os.Getenv(StoragesQty)),
				),
			},
		},
	})
}

const testAccDataSourceStorages = `
data "cloudtemple_backup_storages" "foo" {}
`
