package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	SppServersQty = "BACKUP_SPPSERVER_QTY"
)

func TestAccDataSourceBackupSPPServers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSPPServers,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_servers.foo", "spp_servers.#", os.Getenv(SppServersQty)),
				),
			},
		},
	})
}

const testAccDataSourceBackupSPPServers = `
data "cloudtemple_backup_spp_servers" "foo" {}
`
