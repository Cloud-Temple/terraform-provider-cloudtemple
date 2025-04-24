package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	SppServerId   = "BACKUP_SPPSERVER_ID"
	SppServerName = "BACKUP_SPPSERVER_NAME"
)

func TestAccDataSourceBackupSPPServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceBackupSPPServer, os.Getenv(SppServerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_server.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_server.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_server.foo", "address"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceBackupSPPServerName, os.Getenv(SppServerName)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_server.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_server.foo", "name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_spp_server.foo", "address"),
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
