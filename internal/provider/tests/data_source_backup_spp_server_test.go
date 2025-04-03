package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	SppServerId      = "BACKUP_SPPSERVER_ID"
	SppServerName    = "BACKUP_SPPSERVER_NAME"
	SppServerAddress = "BACKUP_SPPSERVER_ADDRESS"
)

func TestAccDataSourceBackupSPPServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceBackupSPPServer, os.Getenv(SppServerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "id", os.Getenv(SppServerId)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "name", os.Getenv(SppServerName)),
					resource.TestCheckResourceAttr("data.cloudtemple_backup_spp_server.foo", "address", os.Getenv(SppServerAddress)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceBackupSPPServerName, os.Getenv(SppServerName)),
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
  id = "%s"
}
`

const testAccDataSourceBackupSPPServerName = `
data "cloudtemple_backup_spp_server" "foo" {
  name = "%s"
}
`

const testAccDataSourceBackupSPPServerMissing = `
data "cloudtemple_backup_spp_server" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
