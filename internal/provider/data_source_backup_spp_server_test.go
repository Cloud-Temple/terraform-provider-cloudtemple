package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupSPPServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSPPServer,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "id", "a34d230c-dd0f-4fa9-a099-bec7d8609bd4"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "name", "spp01-rec-th3s"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "address", "spp01-rec-th3s.rbackup"),
				),
			},
			{
				Config: testAccDataSourceBackupSPPServerName,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "id", "a34d230c-dd0f-4fa9-a099-bec7d8609bd4"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "name", "spp01-rec-th3s"),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "address", "spp01-rec-th3s.rbackup"),
				),
			},
			{
				Config:      testAccDataSourceBackupSPPServerMissing,
				ExpectError: regexp.MustCompile("failed to find SPP server with id"),
			},
		},
	})
}

const testAccDataSourceBackupSPPServer = `
data "cloudtemple_backup_spp_server" "foo" {
  id = "a34d230c-dd0f-4fa9-a099-bec7d8609bd4"
}
`

const testAccDataSourceBackupSPPServerName = `
data "cloudtemple_backup_spp_server" "foo" {
  name = "spp01-rec-th3s"
}
`

const testAccDataSourceBackupSPPServerMissing = `
data "cloudtemple_backup_spp_server" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
