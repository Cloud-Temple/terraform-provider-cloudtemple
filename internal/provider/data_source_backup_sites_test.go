package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	SitesQty = "BACKUP_SITE_QTY"
)

func TestAccDataSourceBackupSites(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSites,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_sites.foo", "sites.#", os.Getenv(SitesQty)),
				),
			},
		},
	})
}

const testAccDataSourceBackupSites = `
data "cloudtemple_backup_sites" "foo" {}
`
